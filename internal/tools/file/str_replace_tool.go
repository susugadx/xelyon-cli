package file

import (
	"encoding/json"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

// StrReplaceTool はファイルの一部を置換するツール。
type StrReplaceTool struct{}

func (t *StrReplaceTool) Name() string { return "str_replace" }

func (t *StrReplaceTool) Description() string {
	return tools.ToolDescriptions[t.Name()]
}

func (t *StrReplaceTool) Parameters() map[string]interface{} {
	return schemaObject(map[string]interface{}{
		"path":       schemaSinglePathProperty("File path to edit"),
		"old_str":    schemaStringProperty("Exact string to find and replace"),
		"new_str":    schemaStringProperty("New string to replace with"),
		"start_line": schemaStringProperty("Start line number to limit search scope (optional)"),
		"end_line":   schemaStringProperty("End line number to limit search scope (optional)"),
		"edits":      schemaEditsArrayProperty("Batch edits: array of {old_str, new_str} pairs applied sequentially"),
	}, "path")
}

func (t *StrReplaceTool) Run(execCtx tools.ExecutionContext, args map[string]string) (string, *tools.FileChange, error) {
	if args["old_str"] == "" && args["edits"] != "" {
		result, err := executeBatchEditsWithPromptIOAndOptionsResult(execCtx.PromptIO(), execCtx.ConfirmOptions(), args["path"], args["edits"])
		if err != nil {
			return result.message, nil, err
		}
		if !result.ShouldRecordChange() {
			return result.message, nil, nil
		}

		linesAdded, linesRemoved := 0, 0
		var edits []EditEntry
		if json.Unmarshal([]byte(args["edits"]), &edits) == nil {
			for _, edit := range edits {
				linesAdded += countLines(edit.NewStr)
				linesRemoved += countLines(edit.OldStr)
			}
		}

		return result.message, newFileChange(
			args["path"],
			"str_replace",
			"Batch replaced in "+args["path"],
			linesAdded,
			linesRemoved,
		), nil
	}

	result, err := executeStrReplaceWithPromptIOAndOptionsResult(execCtx.PromptIO(), execCtx.ConfirmOptions(), args["path"], args["old_str"], args["new_str"], args["start_line"], args["end_line"])
	if err != nil {
		return result.message, nil, err
	}
	if !result.ShouldRecordChange() {
		return result.message, nil, nil
	}

	return result.message, newFileChange(
		args["path"],
		"str_replace",
		"Replaced in "+args["path"],
		countLines(args["new_str"]),
		countLines(args["old_str"]),
	), nil
}
