package listtool

import (
	"strconv"

	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/file/schema"
)

// ListDirTool はディレクトリ要約ツール。
type ListDirTool struct{}

func (t *ListDirTool) Name() string { return "list_dir" }

func (t *ListDirTool) Description() string {
	return tools.ToolDescription(t.Name())
}

func (t *ListDirTool) Parameters() map[string]interface{} {
	return schema.ListDirParameters()
}

func (t *ListDirTool) Run(execCtx tools.ExecutionContext, args map[string]string) (string, *tools.FileChange, error) {
	depth := 1
	if args["depth"] != "" {
		if n, err := strconv.Atoi(args["depth"]); err == nil {
			depth = n
		}
	}
	return ExecuteWithContext(execCtx, args["path"], depth), nil, nil
}
