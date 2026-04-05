package file

import (
	"strconv"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

// ListDirTool はディレクトリ要約ツール。
type ListDirTool struct{}

func (t *ListDirTool) Name() string { return "list_dir" }

func (t *ListDirTool) Description() string {
	return tools.ToolDescriptions[t.Name()]
}

func (t *ListDirTool) Parameters() map[string]interface{} {
	return schemaObject(map[string]interface{}{
		"path":  schemaSinglePathProperty("Directory path to list"),
		"depth": schemaIntegerProperty("Recursion depth (default: 1, max: 3)"),
	}, "path")
}

func (t *ListDirTool) Run(execCtx tools.ExecutionContext, args map[string]string) (string, *tools.FileChange, error) {
	depth := 1
	if args["depth"] != "" {
		if n, err := strconv.Atoi(args["depth"]); err == nil {
			depth = n
		}
	}
	return ExecuteListDirWithRuntime(execCtx.EffectiveConfig(), execCtx.EffectiveToolCache(), args["path"], depth), nil, nil
}
