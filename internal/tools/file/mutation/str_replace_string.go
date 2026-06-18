package mutation

import (
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/tools/common"
	"github.com/susugadx/xelyon-cli/internal/tools/file/mutation/replaceengine"
)

func executeStringReplacement(ctx fileMutationContext, options common.ConfirmOptions, oldContent, oldStr, newStr string) (fileMutationResult, error) {
	path := ctx.path
	out := ctx.out

	if oldStr == newStr {
		return newErrorMutationResult(fmt.Sprintf("Error: old_str and new_str are identical in %s (no change needed)", path)), nil
	}

	var execution replaceengine.StringExecution
	return executeFileMutationWorkflow(ctx, options, fileMutationWorkflow{
		toolName:       "str_replace",
		confirmMessage: "Apply this replacement? / この置換を適用しますか？",
		preview: func() fileMutationResult {
			execution = replaceengine.BuildStringExecution(oldContent, oldStr, newStr)
			if execution.AttemptedNormalized() && !out.SuppressStdout() {
				out.Yellow.Println("⚠️  Exact match failed, trying normalized whitespace matching...")
			}
			if execution.Failure().HasFailure() {
				return newErrorMutationResult(buildStringReplacementFailure(path, oldContent, oldStr, execution.Failure()))
			}
			showStringReplacementPreview(ctx, oldContent, oldStr, newStr, execution.Plan())
			return fileMutationResult{}
		},
		confirm: buildStrReplaceConfirmHandlers(out, path, strReplaceModeDefault),
		apply: func() (fileMutationResult, error) {
			plan := execution.Plan()
			result := buildAppliedStrReplaceResult(path, plan)
			return applyStringReplaceMutation(ctx, plan.NewContent(), fmt.Sprintf("✅ Replaced in: %s", path), result)
		},
	})
}

func showStringReplacementPreview(ctx fileMutationContext, oldContent, oldStr, newStr string, plan replaceengine.StringPlan) {
	out := ctx.out
	if plan.UsedNormalizedMatch() && !out.SuppressStdout() {
		out.Yellow.Printf("ℹ️  Matched with normalized whitespace (indentation may differ)\n")
		out.Yellow.Printf("   Actual match in file:\n")
		actualMatchedLines := plan.ActualMatchedLines()
		for i, line := range actualMatchedLines {
			if i >= 5 {
				out.Yellow.Printf("   ... (%d more lines)\n", len(actualMatchedLines)-5)
				break
			}
			out.Yellow.Printf("   │ %s\n", line)
		}
		out.Println()
	}
	if out.SuppressStdout() {
		return
	}

	lineDiff := plan.NewLineCount() - plan.OldLineCount()
	absLineDiff := lineDiff
	if absLineDiff < 0 {
		absLineDiff = -absLineDiff
	}
	largeChangeWarning := ""
	if absLineDiff > 100 || plan.OldLineCount() > 100 || plan.NewLineCount() > 100 {
		largeChangeWarning = "  Large change detected. Consider splitting into smaller edits."
	}

	showStrReplaceDiffPreview(ctx, strReplaceDiffPreview{
		targetPath:         ctx.path,
		removedLines:       plan.OldLineCount(),
		addedLines:         plan.NewLineCount(),
		before:             oldStr,
		after:              newStr,
		lineNumOffset:      plan.StartLineForDisplay() - 1,
		largeChangeWarning: largeChangeWarning,
	})

	if newStr != "" && replaceengine.HasNearbyStringDuplicate(oldContent, newStr, plan.MatchStartLine(), plan.MatchEndLine()) {
		out.Yellow.Println("⚠️  Warning: new_str already exists near the replacement (±10 lines, possible duplication)")
	}
}
