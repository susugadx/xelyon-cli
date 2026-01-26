package dev

import (
	"strconv"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

// BashTool executes bash commands
type BashTool struct{}

func (t *BashTool) Name() string { return "bash" }

func (t *BashTool) Description() string {
	return "Executes a shell command. Use for system operations, running scripts, or commands not covered by other tools."
}

func (t *BashTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"command": map[string]interface{}{"type": "string", "description": "Shell command to execute"},
		},
		"required":             []string{"command"},
		"additionalProperties": false,
	}
}

func (t *BashTool) Run(args map[string]string) (string, *tools.FileChange, error) {
	output := ExecuteBash(args["command"])
	return output, nil, nil
}

// RunTestTool runs tests with auto-detection
type RunTestTool struct{}

func (t *RunTestTool) Name() string { return "run_test" }

func (t *RunTestTool) Description() string {
	return "Runs tests for the specified path or the entire project."
}

func (t *RunTestTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{"type": "string", "description": "Test file or directory path (optional)"},
		},
		"additionalProperties": false,
	}
}

func (t *RunTestTool) Run(args map[string]string) (string, *tools.FileChange, error) {
	output := ExecuteRunTest(args["path"])
	return output, nil, nil
}

// FormatTool runs code formatter
type FormatTool struct{}

func (t *FormatTool) Name() string { return "format" }

func (t *FormatTool) Description() string {
	return "Formats source code file (go fmt, prettier, etc.)."
}

func (t *FormatTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path": map[string]interface{}{"type": "string", "description": "File path to format"},
		},
		"required":             []string{"path"},
		"additionalProperties": false,
	}
}

func (t *FormatTool) Run(args map[string]string) (string, *tools.FileChange, error) {
	output, _, err := ExecuteFormat(args["path"])
	return output, nil, err
}

// LintTool runs linter with optional auto-fix
type LintTool struct{}

func (t *LintTool) Name() string { return "lint" }

func (t *LintTool) Description() string {
	return "Runs linter on the specified file with optional auto-fix."
}

func (t *LintTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"path":     map[string]interface{}{"type": "string", "description": "File path to lint"},
			"auto_fix": map[string]interface{}{"type": "string", "description": "Set to 'true' to auto-fix issues"},
		},
		"required":             []string{"path"},
		"additionalProperties": false,
	}
}

func (t *LintTool) Run(args map[string]string) (string, *tools.FileChange, error) {
	output, backupPath, err := ExecuteLint(args["path"], args["auto_fix"])
	if backupPath != "" {
		return output, &tools.FileChange{
			FilePath:   args["path"],
			BackupPath: backupPath,
			Tool:       "lint",
		}, err
	}
	return output, nil, err
}

// HTTPRequestTool executes HTTP requests
type HTTPRequestTool struct{}

func (t *HTTPRequestTool) Name() string { return "http_request" }

func (t *HTTPRequestTool) Description() string {
	return "Executes HTTP requests to external APIs."
}

func (t *HTTPRequestTool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"method":  map[string]interface{}{"type": "string", "description": "HTTP method", "enum": []string{"GET", "POST", "PUT", "DELETE", "PATCH"}},
			"url":     map[string]interface{}{"type": "string", "description": "Request URL"},
			"headers": map[string]interface{}{"type": "string", "description": "Request headers as JSON string"},
			"body":    map[string]interface{}{"type": "string", "description": "Request body"},
			"timeout": map[string]interface{}{"type": "string", "description": "Timeout in seconds (default: 30)"},
		},
		"required":             []string{"method", "url"},
		"additionalProperties": false,
	}
}

func (t *HTTPRequestTool) Run(args map[string]string) (string, *tools.FileChange, error) {
	timeout := 30
	if args["timeout"] != "" {
		if t, err := strconv.Atoi(args["timeout"]); err == nil {
			timeout = t
		}
	}
	output, err := ExecuteHTTPRequest(
		args["method"],
		args["url"],
		args["headers"],
		args["body"],
		timeout,
	)
	if err != nil {
		return "Error: " + err.Error(), nil, nil
	}
	return output, nil, nil
}

// RegisterTools registers all dev tools to the registry
func RegisterTools(registry *tools.Registry) {
	registry.Register(&BashTool{})
	registry.Register(&RunTestTool{})
	registry.Register(&FormatTool{})
	registry.Register(&LintTool{})
	registry.Register(&HTTPRequestTool{})
}

func init() {
	RegisterTools(tools.DefaultRegistry)
}
