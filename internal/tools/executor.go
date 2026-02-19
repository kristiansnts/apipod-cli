package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

type Executor struct {
	workDir  string
	bgShells map[string]*bgShell
	bgMu     sync.Mutex
}

type bgShell struct {
	cmd    *exec.Cmd
	output strings.Builder
	mu     sync.Mutex
}

func NewExecutor(workDir string) *Executor {
	return &Executor{
		workDir:  workDir,
		bgShells: make(map[string]*bgShell),
	}
}

type ToolCall struct {
	ID    string                 `json:"id"`
	Name  string                 `json:"name"`
	Input map[string]interface{} `json:"input"`
}

type ToolResult struct {
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"`
	IsError   bool   `json:"is_error,omitempty"`
}

func (e *Executor) Execute(call ToolCall) ToolResult {
	switch call.Name {
	case "Bash":
		return e.executeBash(call)
	case "Read":
		return e.executeRead(call)
	case "Write":
		return e.executeWrite(call)
	case "Edit":
		return e.executeEdit(call)
	case "MultiEdit":
		return e.executeMultiEdit(call)
	case "Glob":
		return e.executeGlob(call)
	case "Grep":
		return e.executeGrep(call)
	case "BashOutput":
		return e.executeBashOutput(call)
	case "KillBash":
		return e.executeKillBash(call)
	default:
		return ToolResult{ToolUseID: call.ID, Content: fmt.Sprintf("Unknown tool: %s", call.Name), IsError: true}
	}
}

func (e *Executor) resolvePath(p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(e.workDir, p)
}

func (e *Executor) executeBash(call ToolCall) ToolResult {
	command, _ := call.Input["command"].(string)
	if command == "" {
		return ToolResult{ToolUseID: call.ID, Content: "Missing required parameter: command", IsError: true}
	}

	if bg, _ := call.Input["run_in_background"].(bool); bg {
		return e.executeBashBackground(call, command)
	}

	timeout := 120000.0
	if t, ok := call.Input["timeout"].(float64); ok && t > 0 {
		timeout = t
		if timeout > 600000 {
			timeout = 600000
		}
	}

	cmd := exec.Command("bash", "-c", command)
	cmd.Dir = e.workDir

	output, err := cmd.CombinedOutput()
	result := string(output)

	if err != nil {
		if len(result) == 0 {
			result = err.Error()
		}
		return ToolResult{ToolUseID: call.ID, Content: result, IsError: true}
	}

	_ = timeout
	return ToolResult{ToolUseID: call.ID, Content: result}
}

func (e *Executor) executeBashBackground(call ToolCall, command string) ToolResult {
	cmd := exec.Command("bash", "-c", command)
	cmd.Dir = e.workDir

	shell := &bgShell{cmd: cmd}

	stdout, _ := cmd.StdoutPipe()
	cmd.Stderr = cmd.Stdout

	if err := cmd.Start(); err != nil {
		return ToolResult{ToolUseID: call.ID, Content: fmt.Sprintf("Failed to start: %v", err), IsError: true}
	}

	bashID := call.ID
	e.bgMu.Lock()
	e.bgShells[bashID] = shell
	e.bgMu.Unlock()

	go func() {
		buf := make([]byte, 4096)
		for {
			n, err := stdout.Read(buf)
			if n > 0 {
				shell.mu.Lock()
				shell.output.Write(buf[:n])
				shell.mu.Unlock()
			}
			if err != nil {
				break
			}
		}
	}()

	return ToolResult{ToolUseID: call.ID, Content: fmt.Sprintf("Background process started (id: %s)", bashID)}
}

func (e *Executor) executeBashOutput(call ToolCall) ToolResult {
	bashID, _ := call.Input["bash_id"].(string)
	if bashID == "" {
		return ToolResult{ToolUseID: call.ID, Content: "Missing required parameter: bash_id", IsError: true}
	}

	e.bgMu.Lock()
	shell, exists := e.bgShells[bashID]
	e.bgMu.Unlock()

	if !exists {
		return ToolResult{ToolUseID: call.ID, Content: fmt.Sprintf("No background shell: %s", bashID), IsError: true}
	}

	shell.mu.Lock()
	output := shell.output.String()
	shell.output.Reset()
	shell.mu.Unlock()

	if output == "" {
		output = "(no new output)"
	}
	return ToolResult{ToolUseID: call.ID, Content: output}
}

func (e *Executor) executeKillBash(call ToolCall) ToolResult {
	shellID, _ := call.Input["shell_id"].(string)
	if shellID == "" {
		return ToolResult{ToolUseID: call.ID, Content: "Missing required parameter: shell_id", IsError: true}
	}

	e.bgMu.Lock()
	shell, exists := e.bgShells[shellID]
	if exists {
		delete(e.bgShells, shellID)
	}
	e.bgMu.Unlock()

	if !exists {
		return ToolResult{ToolUseID: call.ID, Content: fmt.Sprintf("No background shell: %s", shellID), IsError: true}
	}

	if shell.cmd.Process != nil {
		shell.cmd.Process.Kill()
	}
	return ToolResult{ToolUseID: call.ID, Content: fmt.Sprintf("Shell %s terminated", shellID)}
}

func (e *Executor) executeRead(call ToolCall) ToolResult {
	filePath, _ := call.Input["file_path"].(string)
	if filePath == "" {
		return ToolResult{ToolUseID: call.ID, Content: "Missing required parameter: file_path", IsError: true}
	}

	content, err := os.ReadFile(e.resolvePath(filePath))
	if err != nil {
		return ToolResult{ToolUseID: call.ID, Content: fmt.Sprintf("Error: %v", err), IsError: true}
	}

	lines := strings.Split(string(content), "\n")
	offset, limit := 0, len(lines)

	if v, ok := call.Input["offset"].(float64); ok {
		offset = int(v) - 1
		if offset < 0 {
			offset = 0
		}
	}
	if v, ok := call.Input["limit"].(float64); ok && int(v) > 0 {
		limit = offset + int(v)
	}
	if offset >= len(lines) {
		return ToolResult{ToolUseID: call.ID, Content: "Offset beyond file length", IsError: true}
	}
	if limit > len(lines) {
		limit = len(lines)
	}

	var sb strings.Builder
	for i := offset; i < limit; i++ {
		fmt.Fprintf(&sb, "%5d│%s\n", i+1, lines[i])
	}
	return ToolResult{ToolUseID: call.ID, Content: sb.String()}
}

func (e *Executor) executeWrite(call ToolCall) ToolResult {
	filePath, _ := call.Input["file_path"].(string)
	content, _ := call.Input["content"].(string)
	if filePath == "" {
		return ToolResult{ToolUseID: call.ID, Content: "Missing required parameter: file_path", IsError: true}
	}

	resolved := e.resolvePath(filePath)
	if err := os.MkdirAll(filepath.Dir(resolved), 0755); err != nil {
		return ToolResult{ToolUseID: call.ID, Content: fmt.Sprintf("Error creating dirs: %v", err), IsError: true}
	}

	if err := os.WriteFile(resolved, []byte(content), 0644); err != nil {
		return ToolResult{ToolUseID: call.ID, Content: fmt.Sprintf("Error: %v", err), IsError: true}
	}
	return ToolResult{ToolUseID: call.ID, Content: fmt.Sprintf("Written: %s", filePath)}
}

func (e *Executor) executeEdit(call ToolCall) ToolResult {
	filePath, _ := call.Input["file_path"].(string)
	oldStr, _ := call.Input["old_string"].(string)
	newStr, _ := call.Input["new_string"].(string)

	if filePath == "" || oldStr == "" {
		return ToolResult{ToolUseID: call.ID, Content: "Missing required parameters", IsError: true}
	}

	resolved := e.resolvePath(filePath)
	content, err := os.ReadFile(resolved)
	if err != nil {
		return ToolResult{ToolUseID: call.ID, Content: fmt.Sprintf("Error: %v", err), IsError: true}
	}

	if !strings.Contains(string(content), oldStr) {
		return ToolResult{ToolUseID: call.ID, Content: "String not found in file", IsError: true}
	}

	newContent := strings.Replace(string(content), oldStr, newStr, 1)
	if err := os.WriteFile(resolved, []byte(newContent), 0644); err != nil {
		return ToolResult{ToolUseID: call.ID, Content: fmt.Sprintf("Error: %v", err), IsError: true}
	}
	return ToolResult{ToolUseID: call.ID, Content: fmt.Sprintf("Edited: %s", filePath)}
}

func (e *Executor) executeMultiEdit(call ToolCall) ToolResult {
	filePath, _ := call.Input["file_path"].(string)
	if filePath == "" {
		return ToolResult{ToolUseID: call.ID, Content: "Missing required parameter: file_path", IsError: true}
	}

	editsRaw, ok := call.Input["edits"].([]interface{})
	if !ok || len(editsRaw) == 0 {
		return ToolResult{ToolUseID: call.ID, Content: "Missing required parameter: edits", IsError: true}
	}

	resolved := e.resolvePath(filePath)
	content, err := os.ReadFile(resolved)
	if err != nil {
		return ToolResult{ToolUseID: call.ID, Content: fmt.Sprintf("Error: %v", err), IsError: true}
	}

	text := string(content)
	for i, raw := range editsRaw {
		edit, ok := raw.(map[string]interface{})
		if !ok {
			return ToolResult{ToolUseID: call.ID, Content: fmt.Sprintf("Invalid edit at index %d", i), IsError: true}
		}
		oldStr, _ := edit["old_string"].(string)
		newStr, _ := edit["new_string"].(string)
		if oldStr == "" {
			return ToolResult{ToolUseID: call.ID, Content: fmt.Sprintf("Empty old_string at edit %d", i), IsError: true}
		}
		if !strings.Contains(text, oldStr) {
			return ToolResult{ToolUseID: call.ID, Content: fmt.Sprintf("String not found at edit %d", i), IsError: true}
		}
		if replaceAll, _ := edit["replace_all"].(bool); replaceAll {
			text = strings.ReplaceAll(text, oldStr, newStr)
		} else {
			text = strings.Replace(text, oldStr, newStr, 1)
		}
	}

	if err := os.WriteFile(resolved, []byte(text), 0644); err != nil {
		return ToolResult{ToolUseID: call.ID, Content: fmt.Sprintf("Error: %v", err), IsError: true}
	}
	return ToolResult{ToolUseID: call.ID, Content: fmt.Sprintf("Applied %d edits to %s", len(editsRaw), filePath)}
}

func (e *Executor) executeGlob(call ToolCall) ToolResult {
	pattern, _ := call.Input["pattern"].(string)
	if pattern == "" {
		return ToolResult{ToolUseID: call.ID, Content: "Missing required parameter: pattern", IsError: true}
	}

	resolved := e.resolvePath(pattern)
	matches, err := filepath.Glob(resolved)
	if err != nil {
		return ToolResult{ToolUseID: call.ID, Content: fmt.Sprintf("Error: %v", err), IsError: true}
	}

	if len(matches) == 0 {
		return ToolResult{ToolUseID: call.ID, Content: "No files found"}
	}

	// Make paths relative to workDir
	var relative []string
	for _, m := range matches {
		rel, err := filepath.Rel(e.workDir, m)
		if err != nil {
			relative = append(relative, m)
		} else {
			relative = append(relative, rel)
		}
	}
	return ToolResult{ToolUseID: call.ID, Content: strings.Join(relative, "\n")}
}

func (e *Executor) executeGrep(call ToolCall) ToolResult {
	pattern, _ := call.Input["pattern"].(string)
	if pattern == "" {
		return ToolResult{ToolUseID: call.ID, Content: "Missing required parameter: pattern", IsError: true}
	}

	args := []string{"-rn", pattern}
	if path, ok := call.Input["path"].(string); ok && path != "" {
		args = append(args, e.resolvePath(path))
	} else {
		args = append(args, e.workDir)
	}

	if include, ok := call.Input["include"].(string); ok && include != "" {
		args = append(args, "--include", include)
	}

	cmd := exec.Command("grep", args...)
	output, err := cmd.CombinedOutput()
	if err != nil && len(output) == 0 {
		return ToolResult{ToolUseID: call.ID, Content: "No matches found"}
	}
	return ToolResult{ToolUseID: call.ID, Content: string(output)}
}

func GetToolDefinitions() []json.RawMessage {
	tools := []map[string]interface{}{
		{
			"name":        "Bash",
			"description": "Execute a bash command. Use for running scripts, installing packages, or system operations.",
			"input_schema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"command":     map[string]string{"type": "string", "description": "The bash command to execute"},
					"description": map[string]string{"type": "string", "description": "Short description of what this command does"},
					"timeout":     map[string]interface{}{"type": "number", "description": "Timeout in milliseconds (max 600000)"},
				},
				"required": []string{"command"},
			},
		},
		{
			"name":        "Read",
			"description": "Read the contents of a file. Supports offset and limit for partial reads.",
			"input_schema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"file_path": map[string]string{"type": "string", "description": "Path to the file to read"},
					"offset":    map[string]interface{}{"type": "number", "description": "Line number to start reading from (1-based)"},
					"limit":     map[string]interface{}{"type": "number", "description": "Number of lines to read"},
				},
				"required": []string{"file_path"},
			},
		},
		{
			"name":        "Write",
			"description": "Write content to a file, creating it if it doesn't exist.",
			"input_schema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"file_path": map[string]string{"type": "string", "description": "Path to the file to write"},
					"content":   map[string]string{"type": "string", "description": "Content to write to the file"},
				},
				"required": []string{"file_path", "content"},
			},
		},
		{
			"name":        "Edit",
			"description": "Edit a file by replacing the first occurrence of old_string with new_string.",
			"input_schema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"file_path":  map[string]string{"type": "string", "description": "Path to the file to edit"},
					"old_string": map[string]string{"type": "string", "description": "The string to find and replace"},
					"new_string": map[string]string{"type": "string", "description": "The replacement string"},
				},
				"required": []string{"file_path", "old_string", "new_string"},
			},
		},
		{
			"name":        "MultiEdit",
			"description": "Apply multiple edits to a single file.",
			"input_schema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"file_path": map[string]string{"type": "string", "description": "Path to the file to edit"},
					"edits": map[string]interface{}{
						"type": "array",
						"items": map[string]interface{}{
							"type": "object",
							"properties": map[string]interface{}{
								"old_string":  map[string]string{"type": "string"},
								"new_string":  map[string]string{"type": "string"},
								"replace_all": map[string]interface{}{"type": "boolean"},
							},
							"required": []string{"old_string", "new_string"},
						},
					},
				},
				"required": []string{"file_path", "edits"},
			},
		},
		{
			"name":        "Glob",
			"description": "Find files matching a glob pattern.",
			"input_schema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"pattern": map[string]string{"type": "string", "description": "Glob pattern to match files (e.g. '**/*.go')"},
				},
				"required": []string{"pattern"},
			},
		},
		{
			"name":        "Grep",
			"description": "Search for a pattern in files using grep.",
			"input_schema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"pattern": map[string]string{"type": "string", "description": "Pattern to search for"},
					"path":    map[string]string{"type": "string", "description": "Directory or file to search in"},
					"include": map[string]string{"type": "string", "description": "File pattern to include (e.g. '*.go')"},
				},
				"required": []string{"pattern"},
			},
		},
	}

	var result []json.RawMessage
	for _, t := range tools {
		data, _ := json.Marshal(t)
		result = append(result, data)
	}
	return result
}
