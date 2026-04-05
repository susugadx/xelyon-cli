package file

import (
	"fmt"
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools/common"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func executeStringReplacement(ctx fileMutationContext, options common.ConfirmOptions, oldContent, oldStr, newStr string) (fileMutationResult, error) {
	path := ctx.path
	out := ctx.out

	if oldStr == newStr {
		return newErrorMutationResult(fmt.Sprintf("Error: old_str and new_str are identical in %s (no change needed)", path)), nil
	}

	usedNormalizedMatch := false
	matchStartLine := 0
	matchEndLine := 0
	replacedEndLine := 0
	newContent := ""

	exactMatch := strings.Contains(oldContent, oldStr)
	exactCount := strings.Count(oldContent, oldStr)
	switch {
	case exactMatch && exactCount == 1:
		matchIdx := strings.Index(oldContent, oldStr)
		matchStartLine = 1 + strings.Count(oldContent[:matchIdx], "\n")
		matchEndLine = matchStartLine + strings.Count(oldStr, "\n")
		replacedEndLine = matchStartLine + strings.Count(newStr, "\n")
		newContent = strings.Replace(oldContent, oldStr, newStr, 1)
	case exactMatch && exactCount > 1:
		lines := strings.Split(oldContent, "\n")
		cands := findAllOccurrencesLineRanges(oldContent, oldStr, maxFailureCandidatesToShow)
		return newErrorMutationResult(joinFailureResult(
			fmt.Sprintf("Error: old_str appears %d times in %s (must be unique).", exactCount, path),
			buildCandidateSummary(lines, cands, exactCount),
			"Next: use read_file on one candidate and retry with a more specific old_str; use start_line/end_line for a fixed range; use batch edits to replace all matches.",
		)), nil
	default:
		if !out.SuppressStdout() {
			out.Yellow.Println("⚠️  Exact match failed, trying normalized whitespace matching...")
		}

		found, startIdxNormalized, endIdx := common.FindWithNormalizedWhitespace(oldContent, oldStr)
		if !found {
			lines := strings.Split(oldContent, "\n")
			return newErrorMutationResult(joinFailureResult(
				fmt.Sprintf("Error: old_str not found in %s (tried exact and normalized matching).", path),
				buildHeadPreview(lines, maxFailurePreviewLines),
				"Next: use read_file/search_code to copy the exact text, then retry; use start_line/end_line if you already know the target range.",
			)), nil
		}

		actualOldStr := oldContent[startIdxNormalized : endIdx+1]
		matchStartLine = 1 + strings.Count(oldContent[:startIdxNormalized], "\n")
		matchEndLine = 1 + strings.Count(oldContent[:endIdx], "\n")
		replacedEndLine = matchStartLine + strings.Count(newStr, "\n")
		newContent = oldContent[:startIdxNormalized] + newStr + oldContent[endIdx+1:]
		usedNormalizedMatch = true

		if !out.SuppressStdout() {
			out.Yellow.Printf("ℹ️  Matched with normalized whitespace (indentation may differ)\n")
			out.Yellow.Printf("   Actual match in file:\n")
			matchLines := strings.Split(actualOldStr, "\n")
			for i, line := range matchLines {
				if i >= 5 {
					out.Yellow.Printf("   ... (%d more lines)\n", len(matchLines)-5)
					break
				}
				out.Yellow.Printf("   │ %s\n", line)
			}
			out.Println()
		}
	}

	oldStrLines := strings.Split(oldStr, "\n")
	newStrLines := strings.Split(newStr, "\n")
	lineDiff := len(newStrLines) - len(oldStrLines)

	if !out.SuppressStdout() {
		w := out.StdoutWriter()
		out.Println()
		ui.FileOpHeader(w, "str_replace", path)
		ui.FileOpStatsLine(w, len(oldStrLines), len(newStrLines))

		absLineDiff := lineDiff
		if absLineDiff < 0 {
			absLineDiff = -absLineDiff
		}
		if absLineDiff > 100 || len(oldStrLines) > 100 || len(newStrLines) > 100 {
			out.Yellow.Println("  Large change detected. Consider splitting into smaller edits.")
		}

		startLineForDisplay := 1
		if idx := strings.Index(oldContent, oldStr); idx >= 0 {
			startLineForDisplay = strings.Count(oldContent[:idx], "\n") + 1
		}

		opts := &ui.DiffOptions{
			ContextLines:  ctx.cfg.Diff.ContextLines,
			ShowLineNums:  true,
			InlineMode:    true,
			MaxTotalLines: ctx.cfg.Diff.MaxTotalLines,
			LineNumOffset: startLineForDisplay - 1,
		}
		ui.ShowColoredDiffToWriter(out.StdoutWriter(), oldStr, newStr, opts)
	}

	if newStr != "" && matchStartLine > 0 && !out.SuppressStdout() {
		nearbyStart := matchStartLine - 10
		if nearbyStart < 1 {
			nearbyStart = 1
		}
		nearbyEnd := matchEndLine + 10
		allLines := strings.Split(oldContent, "\n")
		if nearbyEnd > len(allLines) {
			nearbyEnd = len(allLines)
		}

		beforeContent := ""
		if nearbyStart < matchStartLine {
			beforeContent = strings.Join(allLines[nearbyStart-1:matchStartLine-1], "\n")
		}
		afterContent := ""
		if matchEndLine < nearbyEnd {
			afterContent = strings.Join(allLines[matchEndLine:nearbyEnd], "\n")
		}

		if strings.Contains(beforeContent, newStr) || strings.Contains(afterContent, newStr) {
			out.Yellow.Println("⚠️  Warning: new_str already exists near the replacement (±10 lines, possible duplication)")
		}
	}

	if result, ok := confirmFileMutation(ctx, options, "str_replace", "Apply this replacement? / この置換を適用しますか？", mutationConfirmHandlers{
		onComment: func(comment string) fileMutationResult {
			return newCommentMutationResult(buildDeferredStrReplaceResult("[COMMENT]", "", path, comment))
		},
		onCancel: func() fileMutationResult {
			out.Yellow.Println("⚠️  User cancelled the replacement")
			return newCancelledMutationResult(buildDeferredStrReplaceResult("[CANCELLED]", "", path, ""))
		},
	}); !ok {
		return result, nil
	}

	syntaxWarning := validateGoSyntaxForReplace(ctx.absPath, []byte(newContent))
	if syntaxWarning != "" && !out.SuppressStdout() {
		out.Yellow.Printf("%s\n", syntaxWarning)
	}
	if err := os.WriteFile(ctx.absPath, []byte(newContent), 0644); err != nil {
		return newErrorMutationResult(fmt.Sprintf("Error writing file: %v", err)), nil
	}

	out.Green.Printf("✅ Replaced in: %s\n", path)
	result := fmt.Sprintf("Successfully replaced text in %s (lines %d-%d → %d-%d)", path, matchStartLine, matchEndLine, matchStartLine, replacedEndLine)
	if usedNormalizedMatch {
		result = fmt.Sprintf("Successfully replaced text in %s (lines %d-%d → %d-%d, used normalized whitespace matching)", path, matchStartLine, matchEndLine, matchStartLine, replacedEndLine)
	}
	return newAppliedMutationResult(appendSyntaxWarning(result, syntaxWarning)), nil
}
