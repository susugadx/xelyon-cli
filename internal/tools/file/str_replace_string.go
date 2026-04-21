package file

import (
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/tools/common"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func executeStringReplacement(ctx fileMutationContext, options common.ConfirmOptions, oldContent, oldStr, newStr string) (fileMutationResult, error) {
	path := ctx.path
	out := ctx.out

	if oldStr == newStr {
		return newErrorMutationResult(fmt.Sprintf("Error: old_str and new_str are identical in %s (no change needed)", path)), nil
	}

	var execution stringReplacementExecution
	return executeFileMutationWorkflow(ctx, options, fileMutationWorkflow{
		toolName:       "str_replace",
		confirmMessage: "Apply this replacement? / この置換を適用しますか？",
		preview: func() fileMutationResult {
			execution = buildStringReplacementExecution(oldContent, oldStr, newStr)
			if execution.attemptedNormalized && !out.SuppressStdout() {
				out.Yellow.Println("⚠️  Exact match failed, trying normalized whitespace matching...")
			}
			if execution.failure.hasFailure() {
				return newErrorMutationResult(buildStringReplacementFailure(path, oldContent, oldStr, execution.failure))
			}
			showStringReplacementPreview(ctx, oldContent, oldStr, newStr, execution.plan)
			return fileMutationResult{}
		},
		confirm: mutationConfirmHandlers{
			onComment: func(comment string) fileMutationResult {
				return newCommentMutationResult(buildDeferredStrReplaceResult("[COMMENT]", "", path, comment))
			},
			onCancel: func() fileMutationResult {
				out.Yellow.Println("⚠️  User cancelled the replacement")
				return newCancelledMutationResult(buildDeferredStrReplaceResult("[CANCELLED]", "", path, ""))
			},
		},
		apply: func() (fileMutationResult, error) {
			result := buildAppliedStrReplaceResult(path, execution.plan)
			return applyStringReplaceMutation(ctx, execution.plan.newContent, fmt.Sprintf("✅ Replaced in: %s", path), result)
		},
	})
}

func showStringReplacementPreview(ctx fileMutationContext, oldContent, oldStr, newStr string, plan stringReplacementPlan) {
	out := ctx.out
	if plan.usedNormalizedMatch && !out.SuppressStdout() {
		out.Yellow.Printf("ℹ️  Matched with normalized whitespace (indentation may differ)\n")
		out.Yellow.Printf("   Actual match in file:\n")
		for i, line := range plan.actualMatchedLines {
			if i >= 5 {
				out.Yellow.Printf("   ... (%d more lines)\n", len(plan.actualMatchedLines)-5)
				break
			}
			out.Yellow.Printf("   │ %s\n", line)
		}
		out.Println()
	}
	if out.SuppressStdout() {
		return
	}

	w := out.StdoutWriter()
	out.Println()
	ui.FileOpHeader(w, "str_replace", ctx.path)
	ui.FileOpStatsLine(w, plan.oldLineCount, plan.newLineCount)

	lineDiff := plan.newLineCount - plan.oldLineCount
	absLineDiff := lineDiff
	if absLineDiff < 0 {
		absLineDiff = -absLineDiff
	}
	if absLineDiff > 100 || plan.oldLineCount > 100 || plan.newLineCount > 100 {
		out.Yellow.Println("  Large change detected. Consider splitting into smaller edits.")
	}

	opts := &ui.DiffOptions{
		ContextLines:  ctx.cfg.Diff.ContextLines,
		ShowLineNums:  true,
		InlineMode:    true,
		MaxTotalLines: ctx.cfg.Diff.MaxTotalLines,
		LineNumOffset: plan.startLineForDisplay - 1,
	}
	ui.ShowColoredDiffToWriter(out.StdoutWriter(), oldStr, newStr, opts)

	if newStr != "" && hasNearbyStringReplacementDuplicate(oldContent, newStr, plan.matchStartLine, plan.matchEndLine) {
		out.Yellow.Println("⚠️  Warning: new_str already exists near the replacement (±10 lines, possible duplication)")
	}
}
