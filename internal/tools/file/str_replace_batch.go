package file

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

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

	content := oldContent
	for i, edit := range edits {
		if edit.OldStr == "" {
			return newErrorMutationResult(fmt.Sprintf("Error: edits[%d].old_str is empty in %s", i, path)), nil
		}
		if edit.OldStr == edit.NewStr {
			return newErrorMutationResult(fmt.Sprintf("Error: edits[%d] old_str and new_str are identical (no change needed) in %s", i, path)), nil
		}

		count := strings.Count(content, edit.OldStr)
		switch {
		case count == 1:
			content = strings.Replace(content, edit.OldStr, edit.NewStr, 1)
		case count > 1:
			lines := strings.Split(content, "\n")
			cands := findAllOccurrencesLineRanges(content, edit.OldStr, maxFailureCandidatesToShow)
			return newErrorMutationResult(joinFailureResult(
				fmt.Sprintf("Error: edits[%d].old_str appears %d times in %s (must be unique; batch aborted, no changes written).", i, count, path),
				buildCandidateSummary(lines, cands, count),
				fmt.Sprintf("Next: use read_file on one candidate and retry with a more specific edits[%d].old_str; use line-range mode for a fixed block.", i),
			)), nil
		default:
			if !ctx.out.SuppressStdout() {
				ctx.out.Yellow.Printf("⚠️  edits[%d]: Exact match failed, trying normalized whitespace matching...\n", i)
			}
			found, startIdx, endIdx := common.FindWithNormalizedWhitespace(content, edit.OldStr)
			if !found {
				lines := strings.Split(content, "\n")
				return newErrorMutationResult(joinFailureResult(
					fmt.Sprintf("Error: edits[%d].old_str not found in %s (tried exact and normalized matching; batch aborted, no changes written).", i, path),
					buildHeadPreview(lines, maxFailurePreviewLines),
					fmt.Sprintf("Next: use read_file/search_code to copy the exact text for edits[%d].old_str, then retry; split the batch if later edits depend on earlier changes.", i),
				)), nil
			}
			content = content[:startIdx] + edit.NewStr + content[endIdx+1:]
		}
	}

	if content == oldContent {
		return newNoopMutationResult("No changes after applying all edits"), nil
	}

	linesRemoved, linesAdded := batchEditLineStats(edits)
	if !ctx.out.SuppressStdout() {
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
		ui.ShowColoredDiffToWriter(ctx.out.StdoutWriter(), oldContent, content, opts)
	}

	if result, ok := confirmFileMutation(ctx, options, "str_replace", "Apply batch replacement? / バッチ置換を適用しますか？", mutationConfirmHandlers{
		onComment: func(comment string) fileMutationResult {
			return newCommentMutationResult(buildDeferredStrReplaceResult("[COMMENT]", "batch", path, comment))
		},
		onCancel: func() fileMutationResult {
			ctx.out.Yellow.Println("⚠️  User cancelled the batch replacement")
			return newCancelledMutationResult(buildDeferredStrReplaceResult("[CANCELLED]", "batch", path, ""))
		},
	}); !ok {
		return result, nil
	}

	syntaxWarning := validateGoSyntaxForReplace(ctx.absPath, []byte(content))
	if syntaxWarning != "" && !ctx.out.SuppressStdout() {
		ctx.out.Yellow.Printf("%s\n", syntaxWarning)
	}
	if err := os.WriteFile(ctx.absPath, []byte(content), 0644); err != nil {
		return newErrorMutationResult(fmt.Sprintf("Error writing file: %v", err)), nil
	}

	ctx.out.Green.Printf("✅ Applied %d edits to: %s\n", len(edits), path)
	message := fmt.Sprintf("Successfully applied %d edits to %s", len(edits), path)
	return newAppliedMutationResult(appendSyntaxWarning(message, syntaxWarning)), nil
}
