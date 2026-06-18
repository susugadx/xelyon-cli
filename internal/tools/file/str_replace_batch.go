package file

import (
	"fmt"
	"os"

	"github.com/susugadx/xelyon-cli/internal/tools/common"
	"github.com/susugadx/xelyon-cli/internal/uiruntime"
)

type batchEditsExecutionDetails struct {
	result       fileMutationResult
	resolvedPath string
	linesAdded   int
	linesRemoved int
}

// executeBatchEdits は batch edits モードを実行する。
// edits を順番に in-memory 適用し、全成功時のみファイルに書き込む。
// 1つでも失敗したら即 return（ファイル未書き込み = 自動ロールバック）。
func executeBatchEditsWithPromptIOAndOptions(promptIO uiruntime.PromptIO, options common.ConfirmOptions, path, editsJSON string) (string, error) {
	details, err := executeBatchEditsWithPromptIOAndOptionsDetails(promptIO, options, path, editsJSON)
	return details.result.message, err
}

func executeBatchEditsWithPromptIOAndOptionsDetails(promptIO uiruntime.PromptIO, options common.ConfirmOptions, path, editsJSON string) (batchEditsExecutionDetails, error) {
	edits, result := parseBatchEditEntriesResult(editsJSON)
	if result.IsTerminal() {
		return batchEditsExecutionDetails{result: result}, nil
	}
	return executeBatchEditsWithEntriesAndOptionsDetails(promptIO, options, path, edits)
}

func executeBatchEditsWithEntriesAndOptionsDetails(promptIO uiruntime.PromptIO, options common.ConfirmOptions, path string, edits []EditEntry) (batchEditsExecutionDetails, error) {
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

	details := batchEditsExecutionDetails{resolvedPath: ctx.absPath}

	var outcome batchStringReplacementOutcome
	workflowResult, workflowErr := executeFileMutationWorkflow(ctx, options, fileMutationWorkflow{
		toolName:       "str_replace",
		confirmMessage: "Apply batch replacement? / バッチ置換を適用しますか？",
		preview: func() fileMutationResult {
			previewPlan := buildBatchStringReplacementPreviewPlan(path, oldContent, edits)
			outcome = previewPlan.outcome
			for _, editIndex := range outcome.plan.normalizedAttemptedEdits {
				if !ctx.out.SuppressStdout() {
					ctx.out.Yellow.Printf("⚠️  edits[%d]: Exact match failed, trying normalized whitespace matching...\n", editIndex)
				}
			}
			if previewPlan.terminalResult.IsTerminal() {
				return previewPlan.terminalResult
			}
			details.linesAdded, details.linesRemoved = resolveBatchExecutionLineStats(oldContent, outcome.plan.newContent, edits, ctx.out.SuppressStdout())
			showBatchReplacementPreview(ctx, oldContent, outcome.plan.newContent, path, len(edits), details.linesRemoved, details.linesAdded)
			return fileMutationResult{}
		},
		confirm: buildStrReplaceConfirmHandlers(ctx.out, path, strReplaceModeBatch),
		apply: func() (fileMutationResult, error) {
			message := fmt.Sprintf("Successfully applied %d edits to %s", len(edits), path)
			return applyStringReplaceMutation(ctx, outcome.plan.newContent, fmt.Sprintf("✅ Applied %d edits to: %s", len(edits), path), message)
		},
	})
	details.result = workflowResult
	return details, workflowErr
}

func showBatchReplacementPreview(ctx fileMutationContext, oldContent, newContent, path string, editCount, linesRemoved, linesAdded int) {
	if ctx.out.SuppressStdout() {
		return
	}

	lineDiff := linesAdded - linesRemoved
	largeChangeWarning := ""
	if linesRemoved > 100 || linesAdded > 100 || lineDiff > 100 || lineDiff < -100 {
		largeChangeWarning = "  Large change detected. Review the diff carefully."
	}

	showStrReplaceDiffPreview(ctx, strReplaceDiffPreview{
		targetPath:         fmt.Sprintf("%s (batch: %d edits)", path, editCount),
		removedLines:       linesRemoved,
		addedLines:         linesAdded,
		before:             oldContent,
		after:              newContent,
		lineNumOffset:      0,
		largeChangeWarning: largeChangeWarning,
	})
}
