package file

import (
	"fmt"
	"os"

	"github.com/susugadx/xelyon-cli/internal/tools/common"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// ExecuteStrReplaceWithPromptIOAndOptions は確認設定を指定してファイル内の文字列を置換する。
func ExecuteStrReplaceWithPromptIOAndOptions(promptIO ui.PromptIO, options common.ConfirmOptions, path, oldStr, newStr, startLineStr, endLineStr string) (string, error) {
	result, err := executeStrReplaceWithPromptIOAndOptionsResult(promptIO, options, path, oldStr, newStr, startLineStr, endLineStr)
	return result.message, err
}

func executeStrReplaceWithPromptIOAndOptionsResult(promptIO ui.PromptIO, options common.ConfirmOptions, path, oldStr, newStr, startLineStr, endLineStr string) (fileMutationResult, error) {
	ctx, result, err := prepareFileMutation(promptIO, options, path, "path is required")
	if result.message != "" || err != nil {
		return result, err
	}
	if ctx.cfg == nil {
		return fileMutationResult{}, fmt.Errorf("missing confirm options config")
	}

	contentBytes, err := os.ReadFile(ctx.absPath)
	if err != nil {
		return newErrorMutationResult(fmt.Sprintf("Error reading file: %v", err)), nil
	}
	oldContent := string(contentBytes)

	if oldStr == "" {
		return executeLineRangeReplacement(ctx, options, oldContent, newStr, startLineStr, endLineStr)
	}
	return executeStringReplacement(ctx, options, oldContent, oldStr, newStr)
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
		confirm: mutationConfirmHandlers{
			onComment: func(comment string) fileMutationResult {
				return newCommentMutationResult(buildDeferredStrReplaceResult("[COMMENT]", "line range", path, comment))
			},
			onCancel: func() fileMutationResult {
				out.Yellow.Println("⚠️  User cancelled the replacement")
				return newCancelledMutationResult(buildDeferredStrReplaceResult("[CANCELLED]", "line range", path, ""))
			},
		},
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

	w := out.StdoutWriter()
	out.Println()
	ui.FileOpHeader(w, "str_replace", fmt.Sprintf("%s (lines %d-%d)", ctx.path, plan.startLine, plan.endLine))
	ui.FileOpStatsLine(w, plan.oldLineCount, plan.newLineCount)

	opts := &ui.DiffOptions{
		ContextLines:  ctx.cfg.Diff.ContextLines,
		ShowLineNums:  true,
		InlineMode:    true,
		MaxTotalLines: ctx.cfg.Diff.MaxTotalLines,
		LineNumOffset: plan.startLine - 1,
	}
	ui.ShowColoredDiffToWriter(out.StdoutWriter(), plan.beforeRange, newStr, opts)
}
