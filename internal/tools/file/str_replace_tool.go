package file

import (
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
	outcome, err := executeStrReplaceToolRun(execCtx, args)
	if err != nil {
		return outcome.result.message, nil, err
	}
	if !outcome.result.ShouldRecordChange() {
		return outcome.result.message, nil, nil
	}

	return outcome.result.message, newFileChange(
		outcome.displayPath,
		outcome.resolvedPath,
		"str_replace",
		outcome.fileChangeDescription,
		outcome.linesAdded,
		outcome.linesRemoved,
	), nil
}

type strReplaceToolRunOutcome struct {
	result                fileMutationResult
	displayPath           string
	resolvedPath          string
	fileChangeDescription string
	linesAdded            int
	linesRemoved          int
}

func executeStrReplaceToolRun(execCtx tools.ExecutionContext, args map[string]string) (strReplaceToolRunOutcome, error) {
	if args["old_str"] == "" && args["edits"] != "" {
		return executeBatchStrReplaceToolRun(execCtx, args)
	}
	return executeSingleStrReplaceToolRun(execCtx, args)
}

func executeBatchStrReplaceToolRun(execCtx tools.ExecutionContext, args map[string]string) (strReplaceToolRunOutcome, error) {
	details, execErr := executeBatchEditsWithPromptIOAndOptionsDetails(execCtx.PromptIO(), execCtx.ConfirmOptions(), args["path"], args["edits"])

	return strReplaceToolRunOutcome{
		result:                details.result,
		displayPath:           args["path"],
		resolvedPath:          details.resolvedPath,
		fileChangeDescription: "Batch replaced in " + args["path"],
		linesAdded:            details.linesAdded,
		linesRemoved:          details.linesRemoved,
	}, execErr
}

func executeSingleStrReplaceToolRun(execCtx tools.ExecutionContext, args map[string]string) (strReplaceToolRunOutcome, error) {
	details, err := executeStrReplaceWithPromptIOAndOptionsDetails(
		execCtx.PromptIO(),
		execCtx.ConfirmOptions(),
		args["path"],
		args["old_str"],
		args["new_str"],
		args["start_line"],
		args["end_line"],
	)
	return strReplaceToolRunOutcome{
		result:                details.result,
		displayPath:           args["path"],
		resolvedPath:          details.resolvedPath,
		fileChangeDescription: "Replaced in " + args["path"],
		linesAdded:            details.linesAdded,
		linesRemoved:          details.linesRemoved,
	}, err
}
