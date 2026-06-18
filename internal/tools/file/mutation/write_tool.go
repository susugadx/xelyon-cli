package mutation

import (
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/file/schema"
)

// WriteFileTool はファイル全体を書き込むツール。
type WriteFileTool struct{}

func (t *WriteFileTool) Name() string { return "write_file" }

func (t *WriteFileTool) Description() string {
	return tools.ToolDescription(t.Name())
}

func (t *WriteFileTool) Parameters() map[string]interface{} {
	return schema.WriteFileParameters()
}

func (t *WriteFileTool) Run(execCtx tools.ExecutionContext, args map[string]string) (string, *tools.FileChange, error) {
	details, err := executeWriteFileWithPromptIOAndOptionsAndLSPClientDetails(execCtx.PromptIO(), execCtx.ConfirmOptions(), execCtx.EffectiveLSPClient(), args["path"], args["content"])
	if err != nil {
		return details.result.message, nil, err
	}
	change := fileChangeForAppliedMutation(details.result, fileMutationChangeSpec{
		displayPath:  args["path"],
		resolvedPath: details.resolvedPath,
		toolName:     "write_file",
		description:  "Wrote file " + args["path"],
		linesAdded:   countLines(args["content"]),
	})
	return details.result.message, change, nil
}
