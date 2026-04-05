package file

import (
	"fmt"
	"os"
	"strings"

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
	lines := strings.Split(oldContent, "\n")

	hasStart := strings.TrimSpace(startLineStr) != ""
	hasEnd := strings.TrimSpace(endLineStr) != ""
	if !hasStart && !hasEnd {
		return newErrorMutationResult("Error: old_str is required (or provide both start_line and end_line for line-range replacement)"), nil
	}
	if hasStart != hasEnd {
		return newErrorMutationResult("Error: both start_line and end_line are required for line-range replacement (1-indexed inclusive)"), nil
	}

	startLine, endLine, err := parseLineRange(startLineStr, endLineStr)
	if err != nil {
		return newErrorMutationResult(joinFailureResult(
			fmt.Sprintf("Error: invalid line range in %s: %v", path, err),
			"Next: use read_file to confirm start_line/end_line (1-indexed inclusive).",
		)), nil
	}
	if len(lines) == 0 {
		return newErrorMutationResult(fmt.Sprintf("Error: file is empty: %s", path)), nil
	}
	if startLine > len(lines) {
		return newErrorMutationResult(joinFailureResult(
			fmt.Sprintf("Error: start_line is out of range in %s (start_line=%d, file_lines=%d).", path, startLine, len(lines)),
			"Next: use read_file to confirm the target range.",
		)), nil
	}
	if endLine > len(lines) {
		return newErrorMutationResult(joinFailureResult(
			fmt.Sprintf("Error: end_line is out of range in %s (end_line=%d, file_lines=%d).", path, endLine, len(lines)),
			"Next: use read_file to confirm the target range.",
		)), nil
	}

	newStrLines := strings.Split(newStr, "\n")
	newLines := make([]string, 0, len(lines)-(endLine-startLine+1)+len(newStrLines))
	newLines = append(newLines, lines[:startLine-1]...)
	newLines = append(newLines, newStrLines...)
	newLines = append(newLines, lines[endLine:]...)
	newContent := strings.Join(newLines, "\n")

	if newStr != "" && !out.SuppressStdout() {
		nearbyStart := startLine - 10
		if nearbyStart < 1 {
			nearbyStart = 1
		}
		nearbyEnd := endLine + 10
		if nearbyEnd > len(lines) {
			nearbyEnd = len(lines)
		}

		beforeContent := ""
		if nearbyStart < startLine {
			beforeContent = strings.Join(lines[nearbyStart-1:startLine-1], "\n")
		}
		afterContent := ""
		if endLine < nearbyEnd {
			afterContent = strings.Join(lines[endLine:nearbyEnd], "\n")
		}

		if strings.Contains(beforeContent, newStr) || strings.Contains(afterContent, newStr) {
			out.Yellow.Println("⚠️  Warning: new_str already exists near the target range (±10 lines, possible duplication)")
		}
	}

	if !out.SuppressStdout() {
		w := out.StdoutWriter()
		out.Println()
		ui.FileOpHeader(w, "str_replace", fmt.Sprintf("%s (lines %d-%d)", path, startLine, endLine))
		ui.FileOpStatsLine(w, endLine-startLine+1, len(newStrLines))

		beforeStr := strings.Join(lines[startLine-1:endLine], "\n")
		opts := &ui.DiffOptions{
			ContextLines:  ctx.cfg.Diff.ContextLines,
			ShowLineNums:  true,
			InlineMode:    true,
			MaxTotalLines: ctx.cfg.Diff.MaxTotalLines,
			LineNumOffset: startLine - 1,
		}
		ui.ShowColoredDiffToWriter(out.StdoutWriter(), beforeStr, newStr, opts)
	}

	if result, ok := confirmFileMutation(ctx, options, "str_replace", "Apply this replacement? / この置換を適用しますか？", mutationConfirmHandlers{
		onComment: func(comment string) fileMutationResult {
			return newCommentMutationResult(buildDeferredStrReplaceResult("[COMMENT]", "line range", path, comment))
		},
		onCancel: func() fileMutationResult {
			out.Yellow.Println("⚠️  User cancelled the replacement")
			return newCancelledMutationResult(buildDeferredStrReplaceResult("[CANCELLED]", "line range", path, ""))
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

	out.Green.Printf("✅ Replaced lines %d-%d in: %s\n", startLine, endLine, path)
	result := fmt.Sprintf("Successfully replaced lines %d-%d in %s (new range: %d-%d)", startLine, endLine, path, startLine, startLine+len(newStrLines)-1)
	return newAppliedMutationResult(appendSyntaxWarning(result, syntaxWarning)), nil
}
