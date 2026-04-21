package file

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/susugadx/xelyon-cli/internal/tools/common"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// executeBatchEdits は batch edits モードを実行する。
// edits を順番に in-memory 適用し、全成功時のみファイルに書き込む。
// 1つでも失敗したら即 return（ファイル未書き込み = 自動ロールバック）。
func executeBatchEditsWithPromptIOAndOptions(promptIO ui.PromptIO, options common.ConfirmOptions, path, editsJSON string) (string, error) {
	result, err := executeBatchEditsWithPromptIOAndOptionsResult(promptIO, options, path, editsJSON)
	return result.message, err
}

func executeBatchEditsWithPromptIOAndOptionsResult(promptIO ui.PromptIO, options common.ConfirmOptions, path, editsJSON string) (fileMutationResult, error) {
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

	var edits []EditEntry
	if err := json.Unmarshal([]byte(editsJSON), &edits); err != nil {
		return newErrorMutationResult(fmt.Sprintf("Error: invalid edits JSON: %v", err)), nil
	}
	if len(edits) == 0 {
		return newErrorMutationResult("Error: edits array is empty"), nil
	}

	for i, edit := range edits {
		if edit.OldStr == "" {
			return newErrorMutationResult(fmt.Sprintf("Error: edits[%d].old_str is empty in %s", i, path)), nil
		}
		if edit.OldStr == edit.NewStr {
			return newErrorMutationResult(fmt.Sprintf("Error: edits[%d] old_str and new_str are identical (no change needed) in %s", i, path)), nil
		}
	}

	var outcome batchStringReplacementOutcome
	return executeFileMutationWorkflow(ctx, options, fileMutationWorkflow{
		toolName:       "str_replace",
		confirmMessage: "Apply batch replacement? / バッチ置換を適用しますか？",
		preview: func() fileMutationResult {
			outcome = buildBatchStringReplacementOutcome(oldContent, edits)
			for _, editIndex := range outcome.plan.normalizedAttemptedEdits {
				if !ctx.out.SuppressStdout() {
					ctx.out.Yellow.Printf("⚠️  edits[%d]: Exact match failed, trying normalized whitespace matching...\n", editIndex)
				}
			}
			if outcome.failure != nil {
				return newErrorMutationResult(buildBatchStringReplacementFailure(path, *outcome.failure))
			}
			if outcome.plan.newContent == oldContent {
				return newNoopMutationResult("No changes after applying all edits")
			}
			showBatchReplacementPreview(ctx, oldContent, outcome.plan.newContent, path, edits)
			return fileMutationResult{}
		},
		confirm: mutationConfirmHandlers{
			onComment: func(comment string) fileMutationResult {
				return newCommentMutationResult(buildDeferredStrReplaceResult("[COMMENT]", "batch", path, comment))
			},
			onCancel: func() fileMutationResult {
				ctx.out.Yellow.Println("⚠️  User cancelled the batch replacement")
				return newCancelledMutationResult(buildDeferredStrReplaceResult("[CANCELLED]", "batch", path, ""))
			},
		},
		apply: func() (fileMutationResult, error) {
			message := fmt.Sprintf("Successfully applied %d edits to %s", len(edits), path)
			return applyStringReplaceMutation(ctx, outcome.plan.newContent, fmt.Sprintf("✅ Applied %d edits to: %s", len(edits), path), message)
		},
	})
}

func showBatchReplacementPreview(ctx fileMutationContext, oldContent, newContent, path string, edits []EditEntry) {
	if ctx.out.SuppressStdout() {
		return
	}

	linesRemoved, linesAdded := batchEditLineStats(edits)
	w := ctx.out.StdoutWriter()
	ctx.out.Println()
	ui.FileOpHeader(w, "str_replace", fmt.Sprintf("%s (batch: %d edits)", path, len(edits)))
	ui.FileOpStatsLine(w, linesRemoved, linesAdded)

	lineDiff := linesAdded - linesRemoved
	if linesRemoved > 100 || linesAdded > 100 || lineDiff > 100 || lineDiff < -100 {
		ctx.out.Yellow.Println("  Large change detected. Review the diff carefully.")
	}

	opts := &ui.DiffOptions{
		ContextLines:  ctx.cfg.Diff.ContextLines,
		ShowLineNums:  true,
		InlineMode:    true,
		MaxTotalLines: ctx.cfg.Diff.MaxTotalLines,
	}
	ui.ShowColoredDiffToWriter(ctx.out.StdoutWriter(), oldContent, newContent, opts)
}
