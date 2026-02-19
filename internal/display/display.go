package display

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"
)

var (
	Reset       = "\033[0m"
	Bold        = "\033[1m"
	Dim         = "\033[2m"
	Italic      = "\033[3m"
	Underline   = "\033[4m"
	Cyan        = "\033[36m"
	Green       = "\033[32m"
	Yellow      = "\033[33m"
	Red         = "\033[31m"
	Blue        = "\033[34m"
	Magenta     = "\033[35m"
	Gray        = "\033[90m"
	BgGray      = "\033[48;5;236m"
	BgDarkGray  = "\033[48;5;234m"
	White       = "\033[37m"
	BrightCyan  = "\033[96m"
	BrightWhite = "\033[97m"
)

// Lipgloss styles
var (
	headerStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("63")).
			Padding(0, 1).
			Align(lipgloss.Center)

	responseStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240")).
			Padding(0, 1)

	toolStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("241")).
			BorderLeft(true).
			BorderRight(false).
			BorderTop(false).
			BorderBottom(false).
			PaddingLeft(1)

	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("63")).
			Bold(true)

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241"))

	successStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42")).
			Bold(true)

	errorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("196")).
			Bold(true)

	warnStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214")).
			Bold(true)

	promptStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("63")).
			Bold(true)
)

func TermWidth() int {
	w, _, err := term.GetSize(int(os.Stdout.Fd()))
	if err != nil || w <= 0 {
		return 80
	}
	return w
}

func contentWidth() int {
	w := TermWidth()
	if w > 100 {
		w = 100
	}
	return w
}

func Banner(model, cwd string) {
	w := contentWidth()
	dir := filepath.Base(cwd)

	title := titleStyle.Render("◆ apipod-cli") + " " + dimStyle.Render("v0.1.0")
	info := dimStyle.Render(fmt.Sprintf("%s · %s", dir, model))
	tip := dimStyle.Render("Type ") + lipgloss.NewStyle().Foreground(lipgloss.Color("63")).Render("/help") + dimStyle.Render(" for commands")

	content := title + "\n" + info + "\n" + tip

	box := headerStyle.Width(w - 4).Render(content)
	fmt.Println()
	fmt.Println(box)
	fmt.Println()
}

func Prompt() {
	fmt.Printf("%s ", promptStyle.Render("❯"))
}

func AssistantLabel() {
	// Not needed anymore - responses are in panels
}

func Separator() {
	w := contentWidth()
	fmt.Println(dimStyle.Render(strings.Repeat("─", w)))
}

func ThinSeparator() {
	w := contentWidth()
	fmt.Println(dimStyle.Render(strings.Repeat("·", w)))
}

func InfoMessage(msg string) {
	fmt.Println(dimStyle.Render("  " + msg))
}

func ErrorMessage(msg string) {
	fmt.Println(errorStyle.Render("  ✗ " + msg))
}

func SuccessMessage(msg string) {
	fmt.Println(successStyle.Render("  ✓ " + msg))
}

func WarningMessage(msg string) {
	fmt.Println(warnStyle.Render("  ⚠ " + msg))
}

// Spinner for thinking/loading state
type Spinner struct {
	mu      sync.Mutex
	stop    chan struct{}
	stopped bool
	message string
}

var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func NewSpinner(message string) *Spinner {
	s := &Spinner{
		stop:    make(chan struct{}),
		message: message,
	}
	go s.run()
	return s
}

func (s *Spinner) run() {
	i := 0
	for {
		select {
		case <-s.stop:
			fmt.Printf("\r\033[2K")
			return
		default:
			frame := spinnerFrames[i%len(spinnerFrames)]
			fmt.Printf("\r  %s%s %s%s", BrightCyan, frame, s.message, Reset)
			i++
			time.Sleep(80 * time.Millisecond)
		}
	}
}

func (s *Spinner) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.stopped {
		s.stopped = true
		close(s.stop)
		time.Sleep(100 * time.Millisecond)
	}
}

// RenderMarkdown renders streamed text as markdown in a panel
func RenderMarkdown(text string) {
	w := contentWidth()

	renderer, err := glamour.NewTermRenderer(
		glamour.WithAutoStyle(),
		glamour.WithWordWrap(w-6),
	)
	if err != nil {
		// Fallback to plain text
		fmt.Println(text)
		return
	}

	rendered, err := renderer.Render(text)
	if err != nil {
		fmt.Println(text)
		return
	}

	// Trim trailing newlines from glamour output
	rendered = strings.TrimRight(rendered, "\n")

	box := responseStyle.Width(w - 2).Render(rendered)
	fmt.Println(box)
}

func ToolCallStart(name string, input map[string]interface{}) {
	var detail string

	switch name {
	case "Bash":
		if cmd, ok := input["command"].(string); ok {
			lines := strings.Split(cmd, "\n")
			if len(lines) == 1 && len(cmd) < 60 {
				detail = cmd
			} else if len(lines) > 0 {
				detail = lines[0]
				if len(lines) > 1 {
					detail += " ..."
				}
			}
		}
	case "Read":
		if fp, ok := input["file_path"].(string); ok {
			detail = shortenPath(fp)
		}
	case "Write":
		if fp, ok := input["file_path"].(string); ok {
			detail = shortenPath(fp)
		}
	case "Edit", "MultiEdit":
		if fp, ok := input["file_path"].(string); ok {
			detail = shortenPath(fp)
		}
	case "Glob":
		if p, ok := input["pattern"].(string); ok {
			detail = p
		}
	case "Grep":
		if p, ok := input["pattern"].(string); ok {
			detail = p
		}
	}

	icon := toolIcon(name)
	label := warnStyle.Render(icon + " " + name)
	if detail != "" {
		label += " " + dimStyle.Render(detail)
	}
	fmt.Println()
	fmt.Println("  " + label)
}

func toolIcon(name string) string {
	switch name {
	case "Bash", "BashOutput", "KillBash":
		return "❯"
	case "Read":
		return "📄"
	case "Write":
		return "✏️"
	case "Edit", "MultiEdit":
		return "✏️"
	case "Glob", "Grep":
		return "🔍"
	default:
		return "⚡"
	}
}

func shortenPath(path string) string {
	cwd, err := os.Getwd()
	if err != nil {
		return path
	}
	rel, err := filepath.Rel(cwd, path)
	if err != nil || strings.HasPrefix(rel, "..") {
		return path
	}
	return "./" + rel
}

func ToolCallResult(content string, isError bool) {
	lines := strings.Split(strings.TrimRight(content, "\n"), "\n")
	maxLines := 15
	truncated := false
	totalLines := len(lines)
	if len(lines) > maxLines {
		truncated = true
		lines = lines[:maxLines]
	}

	var resultText string
	if isError {
		resultText = errorStyle.Render(strings.Join(lines, "\n"))
	} else {
		resultText = dimStyle.Render(strings.Join(lines, "\n"))
	}
	if truncated {
		resultText += "\n" + dimStyle.Render(fmt.Sprintf("... %d more lines", totalLines-maxLines))
	}

	styled := toolStyle.Render(resultText)
	fmt.Println(styled)
}

func ConfirmPrompt(msg string) bool {
	fmt.Printf("  %s %s ", warnStyle.Render("?"), msg)
	fmt.Printf("%s ", dimStyle.Render("[y/N]"))
	var input string
	fmt.Scanln(&input)
	input = strings.TrimSpace(strings.ToLower(input))
	return input == "y" || input == "yes"
}

func TokenUsage(input, output int) {
	total := input + output
	cost := estimateCost(input, output)
	var info string
	if cost > 0 {
		info = fmt.Sprintf("↳ tokens: %d (%d in, %d out) · ~$%.4f", total, input, output, cost)
	} else {
		info = fmt.Sprintf("↳ tokens: %d (%d in, %d out)", total, input, output)
	}
	fmt.Println(dimStyle.Render("  " + info))
}

func estimateCost(input, output int) float64 {
	inCost := float64(input) / 1_000_000 * 3.0
	outCost := float64(output) / 1_000_000 * 15.0
	return inCost + outCost
}

// StreamingText prints text as it streams in (raw, before final markdown render)
func StreamingText(text string) {
	fmt.Print(text)
}

func StreamingDone() {
	fmt.Println()
}

func LoginInfo(username, plan string) {
	content := successStyle.Render("✓ Authenticated successfully") + "\n\n" +
		dimStyle.Render("Username") + "  " + username + "\n" +
		dimStyle.Render("Plan") + "      " + plan

	box := responseStyle.Width(50).Render(content)
	fmt.Println()
	fmt.Println(box)
	fmt.Println()
}

func LogoutInfo() {
	fmt.Println()
	fmt.Println(successStyle.Render("  ✓ Logged out successfully"))
	fmt.Println()
}

func NotLoggedIn() {
	fmt.Println()
	fmt.Println(warnStyle.Render("  ⚠ Not authenticated"))
	fmt.Println(dimStyle.Render("  Run ") + titleStyle.Render("apipod-cli login") + dimStyle.Render(" to connect your account."))
	fmt.Println()
}

func DeviceCodeDisplay(userCode, verificationURL string) {
	content := lipgloss.NewStyle().Bold(true).Render("🔐 Device Authorization") + "\n\n" +
		dimStyle.Render("Open in browser:") + "\n" +
		lipgloss.NewStyle().Bold(true).Underline(true).Foreground(lipgloss.Color("63")).Render(verificationURL) + "\n\n" +
		dimStyle.Render("Enter this code:") + "\n" +
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("42")).Render("▶  "+userCode+"  ◀")

	box := headerStyle.Width(60).Render(content)
	fmt.Println()
	fmt.Println(box)
	fmt.Println()
}

func DeviceCodeWaiting() {
	fmt.Printf("  %sWaiting for authorization%s", Dim, Reset)
}

func DeviceCodePolling() {
	fmt.Print(".")
}

func WhoamiDisplay(username, plan, baseURL, model, configPath string) {
	content := lipgloss.NewStyle().Bold(true).Render("👤 Account Info") + "\n\n" +
		dimStyle.Render("Username") + "  " + username + "\n" +
		dimStyle.Render("Plan") + "      " + plan + "\n" +
		dimStyle.Render("API URL") + "   " + baseURL + "\n" +
		dimStyle.Render("Model") + "     " + model + "\n" +
		dimStyle.Render("Config") + "    " + configPath

	box := responseStyle.Width(60).Render(content)
	fmt.Println()
	fmt.Println(box)
	fmt.Println()
}

func SlashHelp() {
	commands := []struct{ cmd, desc string }{
		{"/help", "Show this help"},
		{"/clear", "Clear conversation history"},
		{"/model [name]", "Show or change model"},
		{"/compact", "Compact context (clear history)"},
		{"/whoami", "Show current user info"},
		{"/quit", "Exit the session"},
	}
	fmt.Println()
	for _, c := range commands {
		fmt.Printf("  %s  %s\n",
			lipgloss.NewStyle().Foreground(lipgloss.Color("63")).Width(16).Render(c.cmd),
			dimStyle.Render(c.desc))
	}
	fmt.Println()
}

// printBoxLine is now unused but kept for compatibility
func printBoxLine(boxWidth int, content string) {
	vis := stripAnsi(content)
	pad := boxWidth - 4 - len(vis)
	if pad < 0 {
		pad = 0
	}
	fmt.Printf("  %s│%s%s%s%s│%s\n", Dim, Reset, content, strings.Repeat(" ", pad), Dim, Reset)
}

func stripAnsi(s string) string {
	var out strings.Builder
	inEsc := false
	for _, r := range s {
		if r == '\033' {
			inEsc = true
			continue
		}
		if inEsc {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				inEsc = false
			}
			continue
		}
		out.WriteRune(r)
	}
	return out.String()
}
