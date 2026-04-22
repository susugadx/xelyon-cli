package file

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools/common"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

const envBatchExactLineStats = "XELYON_STR_REPLACE_BATCH_EXACT_LINE_STATS"

type batchEditsExecutionDetails struct {
	result       fileMutationResult
	linesAdded   int
	linesRemoved int
}

// executeBatchEdits は batch edits モードを実行する。
// edits を順番に in-memory 適用し、全成功時のみファイルに書き込む。
// 1つでも失敗したら即 return（ファイル未書き込み = 自動ロールバック）。
func executeBatchEditsWithPromptIOAndOptions(promptIO ui.PromptIO, options common.ConfirmOptions, path, editsJSON string) (string, error) {
	result, err := executeBatchEditsWithPromptIOAndOptionsResult(promptIO, options, path, editsJSON)
	return result.message, err
}

func executeBatchEditsWithPromptIOAndOptionsResult(promptIO ui.PromptIO, options common.ConfirmOptions, path, editsJSON string) (fileMutationResult, error) {
	edits, err := parseBatchEditEntries(editsJSON)
	if err != nil {
		return newErrorMutationResult(fmt.Sprintf("Error: invalid edits JSON: %v", err)), nil
	}
	return executeBatchEditsWithEntriesAndOptionsResult(promptIO, options, path, edits)
}

func executeBatchEditsWithEntriesAndOptionsResult(promptIO ui.PromptIO, options common.ConfirmOptions, path string, edits []EditEntry) (fileMutationResult, error) {
	details, err := executeBatchEditsWithEntriesAndOptionsDetails(promptIO, options, path, edits)
	return details.result, err
}

func executeBatchEditsWithEntriesAndOptionsDetails(promptIO ui.PromptIO, options common.ConfirmOptions, path string, edits []EditEntry) (batchEditsExecutionDetails, error) {
	ctx, result, err := prepareFileMutation(promptIO, options, path, "path is required")
	if result.message != "" || err != nil {
		return batchEditsExecutionDetails{result: result}, err
	}
	if ctx.cfg == nil {
		return batchEditsExecutionDetails{}, fmt.Errorf("missing confirm options config")
	}

	contentBytes, err := os.ReadFile(ctx.absPath)
	if err != nil {
		return batchEditsExecutionDetails{result: newErrorMutationResult(fmt.Sprintf("Error reading file: %v", err))}, nil
	}
	oldContent := string(contentBytes)

	if validationErr := validateBatchEditEntries(path, edits); validationErr.IsTerminal() {
		return batchEditsExecutionDetails{result: validationErr}, nil
	}

	details := batchEditsExecutionDetails{}

	var outcome batchStringReplacementOutcome
	workflowResult, workflowErr := executeFileMutationWorkflow(ctx, options, fileMutationWorkflow{
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
			details.linesAdded, details.linesRemoved = resolveBatchExecutionLineStats(ctx, oldContent, outcome.plan.newContent, edits)
			showBatchReplacementPreview(ctx, oldContent, outcome.plan.newContent, path, len(edits), details.linesRemoved, details.linesAdded)
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
	details.result = workflowResult
	return details, workflowErr
}

func resolveBatchExecutionLineStats(ctx fileMutationContext, oldContent, newContent string, edits []EditEntry) (linesAdded, linesRemoved int) {
	linesRemoved, linesAdded = batchEditLineStats(edits)
	if !shouldResolveExactBatchLineStats(ctx) {
		return linesAdded, linesRemoved
	}

	if exactLinesAdded, exactLinesRemoved, exact := resolveBatchDiffLineStats(oldContent, newContent); exact {
		return exactLinesAdded, exactLinesRemoved
	}

	return linesAdded, linesRemoved
}

func shouldResolveExactBatchLineStats(ctx fileMutationContext) bool {
	if isBatchExactLineStatsForced() {
		return true
	}
	return !ctx.out.SuppressStdout()
}

func isBatchExactLineStatsForced() bool {
	switch strings.ToLower(strings.TrimSpace(os.Getenv(envBatchExactLineStats))) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func parseBatchEditEntries(editsJSON string) ([]EditEntry, error) {
	var edits []EditEntry
	if err := json.Unmarshal([]byte(editsJSON), &edits); err != nil {
		return nil, err
	}
	return edits, nil
}

func validateBatchEditEntries(path string, edits []EditEntry) fileMutationResult {
	if len(edits) == 0 {
		return newErrorMutationResult("Error: edits array is empty")
	}

	for i, edit := range edits {
		if edit.OldStr == "" {
			return newErrorMutationResult(fmt.Sprintf("Error: edits[%d].old_str is empty in %s", i, path))
		}
		if edit.OldStr == edit.NewStr {
			return newErrorMutationResult(fmt.Sprintf("Error: edits[%d] old_str and new_str are identical (no change needed) in %s", i, path))
		}
	}

	return fileMutationResult{}
}

func showBatchReplacementPreview(ctx fileMutationContext, oldContent, newContent, path string, editCount, linesRemoved, linesAdded int) {
	if ctx.out.SuppressStdout() {
		return
	}

	w := ctx.out.StdoutWriter()
	ctx.out.Println()
	ui.FileOpHeader(w, "str_replace", fmt.Sprintf("%s (batch: %d edits)", path, editCount))
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
