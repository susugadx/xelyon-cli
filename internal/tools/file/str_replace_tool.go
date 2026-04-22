package file

import (
	"fmt"
	"strings"

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
		outcome.path,
		"str_replace",
		outcome.fileChangeDescription,
		outcome.linesAdded,
		outcome.linesRemoved,
	), nil
}

type strReplaceToolRunOutcome struct {
	result                fileMutationResult
	path                  string
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
	edits, err := parseBatchEditEntries(args["edits"])
	if err != nil {
		return strReplaceToolRunOutcome{
			result: newErrorMutationResult(fmt.Sprintf("Error: invalid edits JSON: %v", err)),
			path:   args["path"],
		}, nil
	}

	details, execErr := executeBatchEditsWithEntriesAndOptionsDetails(execCtx.PromptIO(), execCtx.ConfirmOptions(), args["path"], edits)

	return strReplaceToolRunOutcome{
		result:                details.result,
		path:                  args["path"],
		fileChangeDescription: "Batch replaced in " + args["path"],
		linesAdded:            details.linesAdded,
		linesRemoved:          details.linesRemoved,
	}, execErr
}

func executeSingleStrReplaceToolRun(execCtx tools.ExecutionContext, args map[string]string) (strReplaceToolRunOutcome, error) {
	result, err := executeStrReplaceWithPromptIOAndOptionsResult(
		execCtx.PromptIO(),
		execCtx.ConfirmOptions(),
		args["path"],
		args["old_str"],
		args["new_str"],
		args["start_line"],
		args["end_line"],
	)
	linesAdded, linesRemoved := resolveSingleStrReplaceLineStats(args["old_str"], args["new_str"], args["start_line"], args["end_line"])
	return strReplaceToolRunOutcome{
		result:                result,
		path:                  args["path"],
		fileChangeDescription: "Replaced in " + args["path"],
		linesAdded:            linesAdded,
		linesRemoved:          linesRemoved,
	}, err
}

func resolveSingleStrReplaceLineStats(oldStr, newStr, startLineStr, endLineStr string) (added, removed int) {
	added = countLines(newStr)
	if oldStr != "" {
		return added, countLines(oldStr)
	}
	if strings.TrimSpace(startLineStr) == "" || strings.TrimSpace(endLineStr) == "" {
		return added, 0
	}

	startLine, endLine, err := parseLineRange(startLineStr, endLineStr)
	if err != nil {
		return added, 0
	}
	return added, endLine - startLine + 1
}
