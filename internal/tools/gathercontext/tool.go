package gathercontext

import "github.com/susugadx/xelyon-cli/internal/tools"

// Tool は runtime-owned investigation orchestration を行う高レベルツール。
type Tool struct{}

func (t *Tool) Name() string { return "gather_context" }

func (t *Tool) Description() string {
	return tools.ToolDescription(t.Name())
}

func (t *Tool) Parameters() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"query":       map[string]interface{}{"type": "string", "description": "What code context you need. Can be a symbol, path, path range, locator IDs, or a broader discovery query."},
			"path":        map[string]interface{}{"type": "string", "description": "Optional search scope for discovery queries."},
			"file_filter": map[string]interface{}{"type": "string", "description": "Optional language type (e.g. go, py) or glob filter (e.g. *_test.go) for discovery queries."},
		},
		"required":             []string{"query"},
		"additionalProperties": false,
	}
}

func (t *Tool) Run(execCtx tools.ExecutionContext, args map[string]string) (string, *tools.FileChange, error) {
	result, err := t.RunResult(execCtx, args)
	return result.Output, result.Change, err
}

func (t *Tool) RunResult(execCtx tools.ExecutionContext, args map[string]string) (tools.ToolRunResult, error) {
	req, errResult := parseRequestArgs(args)
	if errResult != "" {
		return tools.ToolRunResult{Output: errResult}, nil
	}
	result := executeRequestResult(execCtx, req)
	return tools.ToolRunResult{
		Output:      formatExecutionResult(result),
		Observation: result.observation,
	}, nil
}
