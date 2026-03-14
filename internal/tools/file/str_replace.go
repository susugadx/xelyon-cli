package file

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	internalast "github.com/susugadx/xelyon-cli/internal/ast"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// EditEntry は batch edits の1エントリ
type EditEntry struct {
	OldStr string `json:"old_str"`
	NewStr string `json:"new_str"`
}

// ExecuteStrReplaceWithPromptIOAndOptions は確認設定を指定してファイル内の文字列を置換する。
func ExecuteStrReplaceWithPromptIOAndOptions(promptIO ui.PromptIO, options common.ConfirmOptions, path, oldStr, newStr, startLineStr, endLineStr string) (string, error) {
	promptIO = ui.NormalizePromptIO(promptIO)
	out := common.NewOutput(promptIO.Out, promptIO.Err)
	cfg := options.Config
	if cfg == nil {
		return "", fmt.Errorf("missing confirm options config")
	}

	if path == "" {
		return "Error: path is required", nil
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), nil
	}

	// ファイルを読み込む
	contentBytes, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Sprintf("Error reading file: %v", err), nil
	}

	oldContent := string(contentBytes)
	var newContent string
	usedNormalizedMatch := false
	var matchStartLine, matchEndLine, replacedEndLine int

	// ==============================
	// 1) Line range replacement mode
	// ==============================
	// old_str が空のときのみレンジ置換を許可（old_str優先）
	if oldStr == "" {
		// 片方欠落の明確化
		hasStart := strings.TrimSpace(startLineStr) != ""
		hasEnd := strings.TrimSpace(endLineStr) != ""

		if !hasStart && !hasEnd {
			return "Error: old_str is required (or provide both start_line and end_line for line-range replacement)", nil
		}
		if hasStart != hasEnd {
			return "Error: both start_line and end_line are required for line-range replacement (1-indexed inclusive)", nil
		}

		startLine, endLine, err := parseLineRange(startLineStr, endLineStr)
		if err != nil {
			return joinFailureResult(
				fmt.Sprintf("Error: invalid line range in %s: %v", path, err),
				"Next: use read_file to confirm start_line/end_line (1-indexed inclusive).",
			), nil
		}

		lines := strings.Split(oldContent, "\n")
		if len(lines) == 0 {
			return fmt.Sprintf("Error: file is empty: %s", path), nil
		}

		if startLine > len(lines) {
			return joinFailureResult(
				fmt.Sprintf("Error: start_line is out of range in %s (start_line=%d, file_lines=%d).", path, startLine, len(lines)),
				"Next: use read_file to confirm the target range.",
			), nil
		}
		if endLine > len(lines) {
			return joinFailureResult(
				fmt.Sprintf("Error: end_line is out of range in %s (end_line=%d, file_lines=%d).", path, endLine, len(lines)),
				"Next: use read_file to confirm the target range.",
			), nil
		}

		// 指定レンジを new_str の行で置き換える
		newStrLines := strings.Split(newStr, "\n")
		newLines := make([]string, 0, len(lines)-(endLine-startLine+1)+len(newStrLines))
		newLines = append(newLines, lines[:startLine-1]...)
		newLines = append(newLines, newStrLines...)
		newLines = append(newLines, lines[endLine:]...)
		newContent = strings.Join(newLines, "\n")

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

		// 確認UI - 変更サマリーを明確に表示
		removed := endLine - startLine + 1
		added := len(newStrLines)
		lineDiff := added - removed

		if !out.SuppressStdout() {
			out.Cyan.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
			out.Cyan.Printf("🔧 str_replace (line range): %s\n", path)
			out.Cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

			out.Yellow.Println("\n📊 Summary / 変更サマリー:")
			out.Printf("   • Range: %d-%d (1-indexed inclusive)\n", startLine, endLine)
			out.Printf("   • Remove %d lines / %d行削除\n", removed, removed)
			out.Printf("   • Add %d lines / %d行追加\n", added, added)
			if lineDiff > 0 {
				out.Green.Printf("   • Net: +%d lines\n", lineDiff)
			} else if lineDiff < 0 {
				out.Red.Printf("   • Net: %d lines\n", lineDiff)
			} else {
				out.Printf("   • Net: 0 lines (same size)\n")
			}

			// Before/After（レンジ部分のみを表示）
			beforeStr := strings.Join(lines[startLine-1:endLine], "\n")
			offset := startLine - 1
			opts := &ui.DiffOptions{
				ContextLines:  cfg.Diff.ContextLines,
				ShowLineNums:  true,
				InlineMode:    true,
				MaxTotalLines: cfg.Diff.MaxTotalLines,
				LineNumOffset: offset,
			}
			ui.ShowColoredDiffToWriter(out.StdoutWriter(), beforeStr, newStr, opts)
		}

		dec := common.ConfirmWithAutoApproveDecisionAndOptions(promptIO, options, "str_replace", "Apply this replacement? / この置換を適用しますか？")
		switch dec.Action {
		case common.ConfirmYes:
			// continue
		case common.ConfirmComment:
			return buildDeferredStrReplaceResult("[COMMENT]", "line range", path, dec.Comment), nil
		default:
			out.Yellow.Println("⚠️  User cancelled the replacement")
			return buildDeferredStrReplaceResult("[CANCELLED]", "line range", path, ""), nil
		}

		syntaxWarning := validateGoSyntaxForReplace(absPath, []byte(newContent))
		if syntaxWarning != "" && !out.SuppressStdout() {
			out.Yellow.Printf("%s\n", syntaxWarning)
		}

		if err := os.WriteFile(absPath, []byte(newContent), 0644); err != nil {
			return fmt.Sprintf("Error writing file: %v", err), nil
		}

		out.Green.Printf("✅ Replaced lines %d-%d in: %s\n", startLine, endLine, path)
		result := fmt.Sprintf("Successfully replaced lines %d-%d in %s (new range: %d-%d)", startLine, endLine, path, startLine, startLine+len(newStrLines)-1)
		return appendSyntaxWarning(result, syntaxWarning), nil
	}

	// ========================
	// 2) String replace mode
	// ========================
	// old_str が非空なら従来どおり（レンジ指定は無視し、互換性を維持）

	// old_str と new_str が同一なら変更不要
	if oldStr == newStr {
		return fmt.Sprintf("Error: old_str and new_str are identical in %s (no change needed)", path), nil
	}

	// まず完全一致を試行
	exactMatch := strings.Contains(oldContent, oldStr)
	exactCount := strings.Count(oldContent, oldStr)

	if exactMatch && exactCount == 1 {
		matchIdx := strings.Index(oldContent, oldStr)
		matchStartLine = 1 + strings.Count(oldContent[:matchIdx], "\n")
		matchEndLine = matchStartLine + strings.Count(oldStr, "\n")
		replacedEndLine = matchStartLine + strings.Count(newStr, "\n")
		newContent = strings.Replace(oldContent, oldStr, newStr, 1)
	} else if exactMatch && exactCount > 1 {
		lines := strings.Split(oldContent, "\n")
		cands := findAllOccurrencesLineRanges(oldContent, oldStr, maxFailureCandidatesToShow)

		return joinFailureResult(
			fmt.Sprintf("Error: old_str appears %d times in %s (must be unique).", exactCount, path),
			buildCandidateSummary(lines, cands, exactCount),
			"Next: use read_file on one candidate and retry with a more specific old_str; use start_line/end_line for a fixed range; use batch edits to replace all matches.",
		), nil
	} else {
		// 完全一致しない → 正規化マッチを試行（従来挙動）
		if !out.SuppressStdout() {
			out.Yellow.Println("⚠️  Exact match failed, trying normalized whitespace matching...")
		}

		found, startIdxNormalized, endIdx := common.FindWithNormalizedWhitespace(oldContent, oldStr)
		if !found {
			lines := strings.Split(oldContent, "\n")
			return joinFailureResult(
				fmt.Sprintf("Error: old_str not found in %s (tried exact and normalized matching).", path),
				buildHeadPreview(lines, maxFailurePreviewLines),
				"Next: use read_file/search_code to copy the exact text, then retry; use start_line/end_line if you already know the target range.",
			), nil
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

	// 確認UI - 変更サマリーを明確に表示
	oldStrLines := strings.Split(oldStr, "\n")
	newStrLines := strings.Split(newStr, "\n")
	lineDiff := len(newStrLines) - len(oldStrLines)

	if !out.SuppressStdout() {
		out.Cyan.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		out.Cyan.Printf("🔧 str_replace: %s\n", path)
		out.Cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

		out.Yellow.Println("\n📊 Summary / 変更サマリー:")
		out.Printf("   • Remove %d lines / %d行削除\n", len(oldStrLines), len(oldStrLines))
		out.Printf("   • Add %d lines / %d行追加\n", len(newStrLines), len(newStrLines))
		if lineDiff > 0 {
			out.Green.Printf("   • Net: +%d lines\n", lineDiff)
		} else if lineDiff < 0 {
			out.Red.Printf("   • Net: %d lines\n", lineDiff)
		} else {
			out.Printf("   • Net: 0 lines (same size)\n")
		}

		// 大規模変更警告
		absLineDiff := lineDiff
		if absLineDiff < 0 {
			absLineDiff = -absLineDiff
		}
		if absLineDiff > 100 || len(oldStrLines) > 100 || len(newStrLines) > 100 {
			out.Red.Println("\n🚨 VERY LARGE CHANGE DETECTED!")
			out.Red.Println("   非常に大きな変更が検出されました。")
			out.Yellow.Println("💡 Consider splitting into multiple smaller str_replace calls.")
			out.Yellow.Println("   複数の小さな str_replace に分割することを検討してください。")
		}

		// diff表示用に、old_str の実ファイル開始行（1-indexed）を算出
		startLineForDisplay := 1
		if idx := strings.Index(oldContent, oldStr); idx >= 0 {
			// 完全一致（または偶然一致）できる場合は、その位置を採用
			startLineForDisplay = strings.Count(oldContent[:idx], "\n") + 1
		}

		offset := startLineForDisplay - 1
		opts := &ui.DiffOptions{
			ContextLines:  cfg.Diff.ContextLines,
			ShowLineNums:  true,
			InlineMode:    true,
			MaxTotalLines: cfg.Diff.MaxTotalLines,
			LineNumOffset: offset,
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

	dec2 := common.ConfirmWithAutoApproveDecisionAndOptions(promptIO, options, "str_replace", "Apply this replacement? / この置換を適用しますか？")
	switch dec2.Action {
	case common.ConfirmYes:
		// continue
	case common.ConfirmComment:
		return buildDeferredStrReplaceResult("[COMMENT]", "", path, dec2.Comment), nil
	default:
		out.Yellow.Println("⚠️  User cancelled the replacement")
		return buildDeferredStrReplaceResult("[CANCELLED]", "", path, ""), nil
	}

	syntaxWarning := validateGoSyntaxForReplace(absPath, []byte(newContent))
	if syntaxWarning != "" && !out.SuppressStdout() {
		out.Yellow.Printf("%s\n", syntaxWarning)
	}

	if err := os.WriteFile(absPath, []byte(newContent), 0644); err != nil {
		return fmt.Sprintf("Error writing file: %v", err), nil
	}

	out.Green.Printf("✅ Replaced in: %s\n", path)
	result := fmt.Sprintf("Successfully replaced text in %s (lines %d-%d → %d-%d)", path, matchStartLine, matchEndLine, matchStartLine, replacedEndLine)
	if usedNormalizedMatch {
		result = fmt.Sprintf("Successfully replaced text in %s (lines %d-%d → %d-%d, used normalized whitespace matching)", path, matchStartLine, matchEndLine, matchStartLine, replacedEndLine)
	}
	return appendSyntaxWarning(result, syntaxWarning), nil
}

type lineRange struct {
	StartLine int
	EndLine   int
}

func parseLineRange(startStr, endStr string) (start, end int, _ error) {
	start64, err := strconv.ParseInt(strings.TrimSpace(startStr), 10, 0)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid start_line: %w", err)
	}
	end64, err := strconv.ParseInt(strings.TrimSpace(endStr), 10, 0)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid end_line: %w", err)
	}
	start = int(start64)
	end = int(end64)

	if start < 1 {
		return 0, 0, fmt.Errorf("start_line must be >= 1")
	}
	if end < start {
		return 0, 0, fmt.Errorf("end_line must be >= start_line")
	}
	return start, end, nil
}

// findAllOccurrencesLineRanges finds up to max occurrences of needle in content and returns their 1-indexed line ranges.
// This is best-effort and used only for error messaging.
func findAllOccurrencesLineRanges(content, needle string, max int) []lineRange {
	if needle == "" || max <= 0 {
		return nil
	}

	var res []lineRange
	searchFrom := 0
	for len(res) < max {
		idx := strings.Index(content[searchFrom:], needle)
		if idx == -1 {
			break
		}
		absIdx := searchFrom + idx
		endIdx := absIdx + len(needle) - 1

		startLine := 1 + strings.Count(content[:absIdx], "\n")
		endLine := 1 + strings.Count(content[:endIdx], "\n")
		res = append(res, lineRange{StartLine: startLine, EndLine: endLine})

		searchFrom = endIdx + 1
		if searchFrom >= len(content) {
			break
		}
	}
	return res
}

const (
	maxFailureCandidatesToShow = 2
	maxFailurePreviewLines     = 3
	failurePreviewLineWidth    = 72
)

func joinFailureResult(parts ...string) string {
	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		filtered = append(filtered, part)
	}
	return strings.Join(filtered, "\n")
}

func buildDeferredStrReplaceResult(status, mode, path, comment string) string {
	header := status + " str_replace"
	if mode != "" {
		header += " (" + mode + ")"
	}
	header += " not applied for " + path + "."
	if strings.TrimSpace(comment) == "" {
		return joinFailureResult(
			header,
			"Next: review with read_file before retrying; do not repeat the same replacement unchanged.",
		)
	}
	return joinFailureResult(
		header,
		"Comment: "+strings.TrimSpace(comment),
		"Next: review with read_file and retry only after user approval.",
	)
}

func buildCandidateSummary(lines []string, cands []lineRange, total int) string {
	if total <= 0 {
		return ""
	}
	shown := common.Min(len(cands), maxFailureCandidatesToShow)
	var b strings.Builder
	fmt.Fprintf(&b, "Candidates: %d total", total)
	if shown > 0 && shown < total {
		fmt.Fprintf(&b, " (showing %d)", shown)
	}
	for i := 0; i < shown; i++ {
		c := cands[i]
		fmt.Fprintf(&b, "\n- lines %d-%d: %s", c.StartLine, c.EndLine, buildInlinePreview(lines, c.StartLine, c.EndLine, 1))
	}
	if total > shown {
		fmt.Fprintf(&b, "\n- ... %d more candidates", total-shown)
	}
	return b.String()
}

func buildHeadPreview(lines []string, limit int) string {
	if len(lines) == 0 || limit <= 0 {
		return ""
	}
	previewCount := common.Min(limit, len(lines))
	parts := make([]string, 0, previewCount)
	for i := 0; i < previewCount; i++ {
		parts = append(parts, fmt.Sprintf("%d:%s", i+1, compactPreviewLine(lines[i])))
	}
	preview := "Preview: " + strings.Join(parts, " | ")
	if len(lines) > previewCount {
		preview += fmt.Sprintf(" | ... +%d more lines", len(lines)-previewCount)
	}
	return preview
}

func buildInlinePreview(lines []string, startLine, endLine, ctx int) string {
	if len(lines) == 0 {
		return ""
	}
	if ctx < 0 {
		ctx = 0
	}
	start := startLine - ctx
	if start < 1 {
		start = 1
	}
	end := endLine + ctx
	if end > len(lines) {
		end = len(lines)
	}
	parts := make([]string, 0, end-start+1)
	for i := start; i <= end; i++ {
		marker := ""
		if i >= startLine && i <= endLine {
			marker = "*"
		}
		parts = append(parts, fmt.Sprintf("%d%s:%s", i, marker, compactPreviewLine(lines[i-1])))
	}
	return strings.Join(parts, " | ")
}

func compactPreviewLine(line string) string {
	line = strings.ReplaceAll(line, "\t", " ")
	line = strings.TrimSpace(line)
	if line == "" {
		return "∅"
	}
	if len(line) <= failurePreviewLineWidth {
		return line
	}
	return line[:failurePreviewLineWidth-3] + "..."
}

// validateGoSyntaxForReplace は置換結果の Go ファイルを AST 検証し、構文エラー警告を返す。
func validateGoSyntaxForReplace(path string, content []byte) string {
	errors := internalast.ValidateSyntax(path, content)
	if len(errors) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("⚠️  AST syntax check found issues after replacement:\n")
	for _, syntaxError := range errors {
		fmt.Fprintf(&b, "   • %s\n", syntaxError.Message)
	}
	b.WriteString("   The replacement was still applied. Consider fixing these issues.")
	return b.String()
}

func appendSyntaxWarning(result, warning string) string {
	if warning == "" {
		return result
	}
	return result + "\n\n" + warning
}

// executeBatchEdits は batch edits モードを実行する。
// edits を順番に in-memory 適用し、全成功時のみファイルに書き込む。
// 1つでも失敗したら即 return（ファイル未書き込み = 自動ロールバック）。
func executeBatchEditsWithPromptIOAndOptions(promptIO ui.PromptIO, options common.ConfirmOptions, path, editsJSON string) (string, error) {
	promptIO = ui.NormalizePromptIO(promptIO)
	out := common.NewOutput(promptIO.Out, promptIO.Err)
	cfg := options.Config
	if cfg == nil {
		return "", fmt.Errorf("missing confirm options config")
	}

	if path == "" {
		return "Error: path is required", nil
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), nil
	}

	contentBytes, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Sprintf("Error reading file: %v", err), nil
	}
	oldContent := string(contentBytes)

	// edits パース
	var edits []EditEntry
	if err := json.Unmarshal([]byte(editsJSON), &edits); err != nil {
		return fmt.Sprintf("Error: invalid edits JSON: %v", err), nil
	}
	if len(edits) == 0 {
		return "Error: edits array is empty", nil
	}

	// edits を順番に in-memory 適用
	content := oldContent
	for i, edit := range edits {
		if edit.OldStr == "" {
			return fmt.Sprintf("Error: edits[%d].old_str is empty in %s", i, path), nil
		}
		if edit.OldStr == edit.NewStr {
			return fmt.Sprintf("Error: edits[%d] old_str and new_str are identical (no change needed) in %s", i, path), nil
		}

		count := strings.Count(content, edit.OldStr)
		switch {
		case count == 1:
			content = strings.Replace(content, edit.OldStr, edit.NewStr, 1)
		case count > 1:
			lines := strings.Split(content, "\n")
			cands := findAllOccurrencesLineRanges(content, edit.OldStr, maxFailureCandidatesToShow)
			return joinFailureResult(
				fmt.Sprintf("Error: edits[%d].old_str appears %d times in %s (must be unique; batch aborted, no changes written).", i, count, path),
				buildCandidateSummary(lines, cands, count),
				fmt.Sprintf("Next: use read_file on one candidate and retry with a more specific edits[%d].old_str; use line-range mode for a fixed block.", i),
			), nil
		default:
			// exact match なし → normalized whitespace fallback
			if !out.SuppressStdout() {
				out.Yellow.Printf("⚠️  edits[%d]: Exact match failed, trying normalized whitespace matching...\n", i)
			}
			found, startIdx, endIdx := common.FindWithNormalizedWhitespace(content, edit.OldStr)
			if !found {
				lines := strings.Split(content, "\n")
				return joinFailureResult(
					fmt.Sprintf("Error: edits[%d].old_str not found in %s (tried exact and normalized matching; batch aborted, no changes written).", i, path),
					buildHeadPreview(lines, maxFailurePreviewLines),
					fmt.Sprintf("Next: use read_file/search_code to copy the exact text for edits[%d].old_str, then retry; split the batch if later edits depend on earlier changes.", i),
				), nil
			}
			content = content[:startIdx] + edit.NewStr + content[endIdx+1:]
		}
	}

	if content == oldContent {
		return "No changes after applying all edits", nil
	}

	// combined diff 表示
	oldLines := strings.Count(oldContent, "\n") + 1
	newLines := strings.Count(content, "\n") + 1
	lineDiff := newLines - oldLines

	if !out.SuppressStdout() {
		out.Cyan.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		out.Cyan.Printf("🔧 str_replace (batch: %d edits): %s\n", len(edits), path)
		out.Cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

		out.Yellow.Println("\n📊 Summary / 変更サマリー:")
		out.Printf("   • Edits: %d\n", len(edits))
		if lineDiff > 0 {
			out.Green.Printf("   • Net: +%d lines\n", lineDiff)
		} else if lineDiff < 0 {
			out.Red.Printf("   • Net: %d lines\n", lineDiff)
		} else {
			out.Printf("   • Net: 0 lines (same size)\n")
		}

		// 大規模変更の警告
		if oldLines > 100 || newLines > 100 || lineDiff > 100 || lineDiff < -100 {
			out.Red.Println("\n⚠️  WARNING: LARGE CHANGE DETECTED")
			out.Red.Println("   This is a significant modification. Please review the diff carefully.")
		}

		opts := &ui.DiffOptions{
			ContextLines:  cfg.Diff.ContextLines,
			ShowLineNums:  true,
			InlineMode:    true,
			MaxTotalLines: cfg.Diff.MaxTotalLines,
		}
		ui.ShowColoredDiffToWriter(out.StdoutWriter(), oldContent, content, opts)
	}

	// 確認
	dec := common.ConfirmWithAutoApproveDecisionAndOptions(promptIO, options, "str_replace", "Apply batch replacement? / バッチ置換を適用しますか？")
	switch dec.Action {
	case common.ConfirmYes:
	// continue
	case common.ConfirmComment:
		return buildDeferredStrReplaceResult("[COMMENT]", "batch", path, dec.Comment), nil
	default:
		out.Yellow.Println("⚠️  User cancelled the batch replacement")
		return buildDeferredStrReplaceResult("[CANCELLED]", "batch", path, ""), nil
	}

	syntaxWarning := validateGoSyntaxForReplace(absPath, []byte(content))
	if syntaxWarning != "" && !out.SuppressStdout() {
		out.Yellow.Printf("%s\n", syntaxWarning)
	}

	if err := os.WriteFile(absPath, []byte(content), 0644); err != nil {
		return fmt.Sprintf("Error writing file: %v", err), nil
	}

	out.Green.Printf("✅ Applied %d edits to: %s\n", len(edits), path)
	result := fmt.Sprintf("Successfully applied %d edits to %s", len(edits), path)
	return appendSyntaxWarning(result, syntaxWarning), nil
}
