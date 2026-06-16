package file

import "github.com/susugadx/xelyon-cli/internal/tools"

// DeleteFileTool はファイル削除ツール。
type DeleteFileTool struct{}

func (t *DeleteFileTool) Name() string { return "delete_file" }

func (t *DeleteFileTool) Description() string {
	return tools.ToolDescription(t.Name())
}

func (t *DeleteFileTool) Parameters() map[string]interface{} {
	return schemaObject(map[string]interface{}{
		"path": schemaSinglePathProperty("File path to delete"),
	}, "path")
}

func (t *DeleteFileTool) Run(execCtx tools.ExecutionContext, args map[string]string) (string, *tools.FileChange, error) {
	details, err := executeDeleteFileWithPromptIOAndOptionsAndLSPClientDetails(execCtx.PromptIO(), execCtx.ConfirmOptions(), execCtx.EffectiveLSPClient(), args["path"])
	if err != nil {
		return details.result.message, nil, err
	}
	change := fileChangeForAppliedMutation(details.result, fileMutationChangeSpec{
		displayPath:  args["path"],
		resolvedPath: details.resolvedPath,
		toolName:     "delete_file",
		description:  "Deleted file " + args["path"],
	})
	return details.result.message, change, nil
}
