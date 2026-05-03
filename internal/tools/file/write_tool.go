package file

import "github.com/susugadx/xelyon-cli/internal/tools"

// WriteFileTool はファイル全体を書き込むツール。
type WriteFileTool struct{}

func (t *WriteFileTool) Name() string { return "write_file" }

func (t *WriteFileTool) Description() string {
	return tools.ToolDescriptions[t.Name()]
}

func (t *WriteFileTool) Parameters() map[string]interface{} {
	return schemaObject(map[string]interface{}{
		"path":    schemaSinglePathProperty("File path to write to"),
		"content": schemaStringProperty("Content to write to the file"),
	}, "path", "content")
}

func (t *WriteFileTool) Run(execCtx tools.ExecutionContext, args map[string]string) (string, *tools.FileChange, error) {
	details, err := executeWriteFileWithPromptIOAndOptionsAndLSPClientDetails(execCtx.PromptIO(), execCtx.ConfirmOptions(), execCtx.EffectiveLSPClient(), args["path"], args["content"])
	if err != nil {
		return details.result.message, nil, err
	}
	if !details.result.ShouldRecordChange() {
		return details.result.message, nil, nil
	}
	return details.result.message, newFileChange(
		args["path"],
		details.resolvedPath,
		"write_file",
		"Wrote file "+args["path"],
		countLines(args["content"]),
		0,
	), nil
}
