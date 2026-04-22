package file

import (
	"fmt"
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools/common"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

type strReplaceExecutionDetails struct {
	result       fileMutationResult
	linesAdded   int
	linesRemoved int
}

// ExecuteStrReplaceWithPromptIOAndOptions は確認設定を指定してファイル内の文字列を置換する。
func ExecuteStrReplaceWithPromptIOAndOptions(promptIO ui.PromptIO, options common.ConfirmOptions, path, oldStr, newStr, startLineStr, endLineStr string) (string, error) {
	details, err := executeStrReplaceWithPromptIOAndOptionsDetails(promptIO, options, path, oldStr, newStr, startLineStr, endLineStr)
	return details.result.message, err
}

func executeStrReplaceWithPromptIOAndOptionsDetails(promptIO ui.PromptIO, options common.ConfirmOptions, path, oldStr, newStr, startLineStr, endLineStr string) (strReplaceExecutionDetails, error) {
	linesAdded, linesRemoved := resolveSingleStrReplaceLineStats(oldStr, newStr, startLineStr, endLineStr)
	details := strReplaceExecutionDetails{
		linesAdded:   linesAdded,
		linesRemoved: linesRemoved,
	}

	ctx, result, err := prepareFileMutation(promptIO, options, path, "path is required")
	if result.message != "" || err != nil {
		details.result = result
		return details, err
	}
	if ctx.cfg == nil {
		return strReplaceExecutionDetails{}, fmt.Errorf("missing confirm options config")
	}

	contentBytes, err := os.ReadFile(ctx.absPath)
	if err != nil {
		details.result = newErrorMutationResult(fmt.Sprintf("Error reading file: %v", err))
		return details, nil
	}
	oldContent := string(contentBytes)

	if oldStr == "" {
		details.result, err = executeLineRangeReplacement(ctx, options, oldContent, newStr, startLineStr, endLineStr)
		return details, err
	}
	details.result, err = executeStringReplacement(ctx, options, oldContent, oldStr, newStr)
	return details, err
}

func executeLineRangeReplacement(ctx fileMutationContext, options common.ConfirmOptions, oldContent, newStr, startLineStr, endLineStr string) (fileMutationResult, error) {
	path := ctx.path
	out := ctx.out
	var execution lineRangeReplacementExecution
	return executeFileMutationWorkflow(ctx, options, fileMutationWorkflow{
		toolName:       "str_replace",
		confirmMessage: "Apply this replacement? / この置換を適用しますか？",
		preview: func() fileMutationResult {
			execution = buildLineRangeReplacementExecution(oldContent, newStr, startLineStr, endLineStr)
			if execution.failure.hasFailure() {
				return newErrorMutationResult(buildLineRangeReplacementFailure(path, execution.failure))
			}
			showLineRangeReplacementPreview(ctx, newStr, execution.plan)
			return fileMutationResult{}
		},
		confirm: buildStrReplaceConfirmHandlers(out, path, strReplaceModeLineRange),
		apply: func() (fileMutationResult, error) {
			result := buildAppliedLineRangeStrReplaceResult(path, execution.plan)
			return applyStringReplaceMutation(
				ctx,
				execution.plan.newContent,
				fmt.Sprintf("✅ Replaced lines %d-%d in: %s", execution.plan.startLine, execution.plan.endLine, path),
				result,
			)
		},
	})
}

func showLineRangeReplacementPreview(ctx fileMutationContext, newStr string, plan lineRangeReplacementPlan) {
	out := ctx.out
	if newStr != "" && !out.SuppressStdout() && hasNearbyLineRangeReplacementDuplicate(plan.lines, newStr, plan.startLine, plan.endLine) {
		out.Yellow.Println("⚠️  Warning: new_str already exists near the target range (±10 lines, possible duplication)")
	}
	if out.SuppressStdout() {
		return
	}

	showStrReplaceDiffPreview(ctx, strReplaceDiffPreview{
		targetPath:    fmt.Sprintf("%s (lines %d-%d)", ctx.path, plan.startLine, plan.endLine),
		removedLines:  plan.oldLineCount,
		addedLines:    plan.newLineCount,
		before:        plan.beforeRange,
		after:         newStr,
		lineNumOffset: plan.startLine - 1,
	})
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
