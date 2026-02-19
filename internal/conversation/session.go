package conversation

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"

	"github.com/rpay/apipod-cli/internal/client"
	"github.com/rpay/apipod-cli/internal/display"
	"github.com/rpay/apipod-cli/internal/tools"
)

const maxToolIterations = 25

type Session struct {
	client   *client.Client
	executor *tools.Executor
	model    string
	messages []client.Message
	system   string
}

func NewSession(c *client.Client, model, workDir string) *Session {
	cwd, _ := os.Getwd()
	if workDir != "" {
		cwd = workDir
	}

	system := buildSystemPrompt(cwd)

	return &Session{
		client:   c,
		executor: tools.NewExecutor(cwd),
		model:    model,
		messages: []client.Message{},
		system:   system,
	}
}

func buildSystemPrompt(cwd string) string {
	var sb strings.Builder
	sb.WriteString("You are an agentic coding assistant running in the user's terminal via apipod-cli.\n")
	sb.WriteString("You help with software engineering tasks: writing code, debugging, running commands, and explaining code.\n\n")
	sb.WriteString("Guidelines:\n")
	sb.WriteString("- Be concise and direct\n")
	sb.WriteString("- Use tools to explore the codebase before making changes\n")
	sb.WriteString("- Make minimal, surgical changes\n")
	sb.WriteString("- Run tests/builds after changes when possible\n")
	sb.WriteString("- Do not add unnecessary comments to code\n\n")

	sb.WriteString(fmt.Sprintf("Working directory: %s\n", cwd))
	sb.WriteString(fmt.Sprintf("Platform: %s/%s\n", runtime.GOOS, runtime.GOARCH))

	if info, err := os.ReadDir(cwd); err == nil {
		var files []string
		for _, f := range info {
			if !strings.HasPrefix(f.Name(), ".") {
				files = append(files, f.Name())
			}
		}
		if len(files) > 0 {
			sb.WriteString(fmt.Sprintf("Directory contents: %s\n", strings.Join(files, ", ")))
		}
	}

	return sb.String()
}

func (s *Session) SendMessage(userInput string) error {
	s.messages = append(s.messages, client.Message{
		Role:    "user",
		Content: userInput,
	})

	return s.runLoop()
}

func (s *Session) runLoop() error {
	toolDefs := s.getToolDefinitions()

	for i := 0; i < maxToolIterations; i++ {
		req := &client.MessagesRequest{
			Model:    s.model,
			Messages: s.messages,
			System:   s.system,
			Tools:    toolDefs,
		}

		spinner := display.NewSpinner("Thinking...")
		var textAccumulator strings.Builder
		streaming := false

		cb := &client.StreamCallback{
			OnText: func(text string) {
				spinner.Stop()
				if !streaming {
					streaming = true
				}
				textAccumulator.WriteString(text)
				// Show raw streaming text as it comes in
				display.StreamingText(text)
			},
			OnToolUseStart: func(id, name string) {
				spinner.Stop()
			},
			OnError: func(err error) {
				spinner.Stop()
				display.ErrorMessage(err.Error())
			},
		}

		resp, err := s.client.SendMessageStream(req, cb)
		spinner.Stop()

		// If we streamed text, render it as formatted markdown
		if streaming && textAccumulator.Len() > 0 {
			// Clear the raw streamed text and replace with markdown
			fmt.Print("\r\033[2K")
			rawText := textAccumulator.String()
			rawLines := strings.Count(rawText, "\n")
			for i := 0; i < rawLines; i++ {
				fmt.Print("\033[A\033[2K")
			}
			fmt.Print("\r")
			display.RenderMarkdown(rawText)
		}

		if err != nil {
			return fmt.Errorf("API error: %w", err)
		}

		hasToolUse := false
		var toolResults []interface{}

		for _, block := range resp.Content {
			if block.Type == "tool_use" {
				hasToolUse = true

				var input map[string]interface{}
				if err := json.Unmarshal(block.Input, &input); err != nil {
					input = map[string]interface{}{}
				}

				display.ToolCallStart(block.Name, input)

				if needsConfirmation(block.Name, input) {
					if !display.ConfirmPrompt(fmt.Sprintf("Allow %s?", block.Name)) {
						toolResults = append(toolResults, map[string]interface{}{
							"type":        "tool_result",
							"tool_use_id": block.ID,
							"content":     "User denied this operation",
							"is_error":    true,
						})
						continue
					}
				}

				result := s.executor.Execute(tools.ToolCall{
					ID:    block.ID,
					Name:  block.Name,
					Input: input,
				})

				display.ToolCallResult(result.Content, result.IsError)

				toolResults = append(toolResults, map[string]interface{}{
					"type":        "tool_result",
					"tool_use_id": result.ToolUseID,
					"content":     result.Content,
					"is_error":    result.IsError,
				})
			}
		}

		// Add assistant response to history
		var contentBlocks []interface{}
		for _, block := range resp.Content {
			switch block.Type {
			case "text":
				contentBlocks = append(contentBlocks, map[string]interface{}{
					"type": "text",
					"text": block.Text,
				})
			case "tool_use":
				contentBlocks = append(contentBlocks, map[string]interface{}{
					"type":  "tool_use",
					"id":    block.ID,
					"name":  block.Name,
					"input": json.RawMessage(block.Input),
				})
			}
		}
		s.messages = append(s.messages, client.Message{
			Role:    "assistant",
			Content: contentBlocks,
		})

		if !hasToolUse {
			display.TokenUsage(resp.Usage.InputTokens, resp.Usage.OutputTokens)
			break
		}

		// Add tool results as user message
		s.messages = append(s.messages, client.Message{
			Role:    "user",
			Content: toolResults,
		})
	}

	return nil
}

func (s *Session) getToolDefinitions() []client.ToolDefinition {
	raw := tools.GetToolDefinitions()
	var defs []client.ToolDefinition
	for _, r := range raw {
		var def client.ToolDefinition
		if err := json.Unmarshal(r, &def); err == nil {
			defs = append(defs, def)
		}
	}
	return defs
}

func (s *Session) Clear() {
	s.messages = nil
	display.SuccessMessage("Conversation cleared")
}

func needsConfirmation(toolName string, input map[string]interface{}) bool {
	switch toolName {
	case "Bash":
		return true
	case "Write":
		return true
	case "Edit", "MultiEdit":
		return true
	default:
		return false
	}
}
