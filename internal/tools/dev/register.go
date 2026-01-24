package dev

import (
	"strconv"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

// BashTool executes bash commands
type BashTool struct{}

func (t *BashTool) Name() string { return "bash" }

func (t *BashTool) Run(args map[string]string) (string, *tools.FileChange, error) {
	output := ExecuteBash(args["command"])
	return output, nil, nil
}

// RunTestTool runs tests with auto-detection
type RunTestTool struct{}

func (t *RunTestTool) Name() string { return "run_test" }

func (t *RunTestTool) Run(args map[string]string) (string, *tools.FileChange, error) {
	output := ExecuteRunTest(args["path"])
	return output, nil, nil
}

// FormatTool runs code formatter
type FormatTool struct{}

func (t *FormatTool) Name() string { return "format" }

func (t *FormatTool) Run(args map[string]string) (string, *tools.FileChange, error) {
	output, _, err := ExecuteFormat(args["path"])
	return output, nil, err
}

// LintTool runs linter with optional auto-fix
type LintTool struct{}

func (t *LintTool) Name() string { return "lint" }

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
