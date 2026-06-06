package agent

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type ToolExecutor struct {
	cwd string
}

func NewToolExecutor(cwd string) *ToolExecutor {
	return &ToolExecutor{cwd: cwd}
}

func DefaultTools() []ToolDefinition {
	return []ToolDefinition{
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "read_file",
				Description: "Read the contents of a file at the given path",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"file_path": map[string]any{
							"type":        "string",
							"description": "Absolute path or path relative to the working directory",
						},
					},
					"required": []string{"file_path"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "write_file",
				Description: "Write content to a file, creating or overwriting it",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"file_path": map[string]any{
							"type":        "string",
							"description": "Absolute path or path relative to the working directory",
						},
						"content": map[string]any{
							"type":        "string",
							"description": "The full content to write to the file",
						},
					},
					"required": []string{"file_path", "content"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "bash",
				Description: "Execute a shell command (30s timeout, use sh -c)",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"command": map[string]any{
							"type":        "string",
							"description": "The shell command to execute",
						},
						"timeout": map[string]any{
							"type":        "number",
							"description": "Timeout in seconds (default 30)",
						},
					},
					"required": []string{"command"},
				},
			},
		},
		{
			Type: "function",
			Function: ToolFunction{
				Name:        "glob",
				Description: "Find files matching a glob pattern",
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"pattern": map[string]any{
							"type":        "string",
							"description": "Glob pattern (e.g. **/*.go, src/**/*.ts)",
						},
					},
					"required": []string{"pattern"},
				},
			},
		},
	}
}

type ToolResult struct {
	Output string
	Error  string
}

func (e *ToolExecutor) Execute(ctx context.Context, name string, args map[string]any) ToolResult {
	switch name {
	case "read_file":
		return e.readFile(args)
	case "write_file":
		return e.writeFile(args)
	case "bash":
		return e.bash(ctx, args)
	case "glob":
		return e.glob(args)
	default:
		return ToolResult{Error: fmt.Sprintf("unknown tool: %s", name)}
	}
}

func (e *ToolExecutor) resolvePath(path string) (string, error) {
	base := filepath.Clean(e.cwd)
	resolved := path
	if !filepath.IsAbs(resolved) {
		resolved = filepath.Join(base, resolved)
	}
	cleaned := filepath.Clean(resolved)
	if !strings.HasPrefix(cleaned, base+string(filepath.Separator)) && cleaned != base {
		return "", fmt.Errorf("path %q escapes working directory", path)
	}
	return cleaned, nil
}

func (e *ToolExecutor) readFile(args map[string]any) ToolResult {
	path, _ := args["file_path"].(string)
	if path == "" {
		return ToolResult{Error: "file_path is required"}
	}
	path, err := e.resolvePath(path)
	if err != nil {
		return ToolResult{Error: err.Error()}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return ToolResult{Error: fmt.Sprintf("read file: %s", err)}
	}

	return ToolResult{Output: string(data)}
}

func (e *ToolExecutor) writeFile(args map[string]any) ToolResult {
	path, _ := args["file_path"].(string)
	content, _ := args["content"].(string)
	if path == "" {
		return ToolResult{Error: "file_path is required"}
	}
	path, err := e.resolvePath(path)
	if err != nil {
		return ToolResult{Error: err.Error()}
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return ToolResult{Error: fmt.Sprintf("mkdir: %s", err)}
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return ToolResult{Error: fmt.Sprintf("write file: %s", err)}
	}

	return ToolResult{Output: fmt.Sprintf("wrote %d bytes to %s", len(content), path)}
}

func (e *ToolExecutor) bash(ctx context.Context, args map[string]any) ToolResult {
	command, _ := args["command"].(string)
	if command == "" {
		return ToolResult{Error: "command is required"}
	}

	timeoutSec := 30
	if t, ok := args["timeout"].(float64); ok && t > 0 {
		timeoutSec = int(t)
	}

	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	cmd.Dir = e.cwd

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	var out strings.Builder
	if stdout.Len() > 0 {
		out.WriteString(stdout.String())
	}
	if stderr.Len() > 0 {
		if out.Len() > 0 {
			out.WriteString("\n")
		}
		out.WriteString(stderr.String())
	}

	result := ToolResult{Output: out.String()}
	if err != nil {
		result.Error = fmt.Sprintf("exit: %s", err)
	}

	return result
}

func (e *ToolExecutor) glob(args map[string]any) ToolResult {
	pattern, _ := args["pattern"].(string)
	if pattern == "" {
		return ToolResult{Error: "pattern is required"}
	}

	resolved, err := e.resolvePath(pattern)
	if err != nil {
		return ToolResult{Error: err.Error()}
	}

	matches, err := filepath.Glob(resolved)
	if err != nil {
		return ToolResult{Error: fmt.Sprintf("glob: %s", err)}
	}

	if len(matches) == 0 {
		return ToolResult{Output: "no matches found"}
	}

	// filter out matches that escape cwd
	base := filepath.Clean(e.cwd)
	var safe []string
	for _, m := range matches {
		cleaned := filepath.Clean(m)
		if (strings.HasPrefix(cleaned, base+string(filepath.Separator)) || cleaned == base) && !strings.Contains(cleaned, "..") {
			safe = append(safe, m)
		}
	}

	if len(safe) == 0 {
		return ToolResult{Output: "no matches found"}
	}

	return ToolResult{Output: strings.Join(safe, "\n")}
}
