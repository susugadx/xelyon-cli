package mutation

import (
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/file/schema"
)

// StrReplaceTool はファイルの一部を置換するツール。
type StrReplaceTool struct{}

func (t *StrReplaceTool) Name() string { return "str_replace" }

func (t *StrReplaceTool) Description() string {
	return tools.ToolDescription(t.Name())
}

func (t *StrReplaceTool) Parameters() map[string]interface{} {
	return schema.StrReplaceParameters()
}

func (t *StrReplaceTool) Run(execCtx tools.ExecutionContext, args map[string]string) (string, *tools.FileChange, error) {
	outcome, err := executeStrReplaceToolRun(execCtx, args)
	if err != nil {
		return outcome.result.message, nil, err
	}
	change := fileChangeForAppliedMutation(outcome.result, fileMutationChangeSpec{
		displayPath:  outcome.displayPath,
		resolvedPath: outcome.resolvedPath,
		toolName:     "str_replace",
		description:  outcome.fileChangeDescription,
		linesAdded:   outcome.linesAdded,
		linesRemoved: outcome.linesRemoved,
	})
	return outcome.result.message, change, nil
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
