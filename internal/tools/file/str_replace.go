package file

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
	"github.com/susugadx/xelyon-cli/internal/tools/lsp"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// ExecuteStrReplace はファイル内の文字列を置換
//
// Behavior:
// - old_str が非空の場合: 既存の文字列置換モード（old_str優先、従来挙動を維持）
// - old_str が空 かつ start_line/end_line が両方指定の場合: 行レンジ置換モード
//
// NOTE:
// - 行レンジは 1-indexed inclusive（start_line=1 は先頭行）
// - start/end は範囲外の場合エラー（delete_lines とは異なりクランプしない）
func ExecuteStrReplace(path, oldStr, newStr, startLineStr, endLineStr string) (string, error) {
	if path == "" {
		return "Error: path is required", nil
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Sprintf("Error: %v", err), nil
	}

	// Read-Before-Write guard: read_file 前の str_replace をブロック
	if !tools.GlobalReadTracker.IsRead(absPath) {
		return fmt.Sprintf("Error: You must read_file before str_replace. Run read_file(path=\"%s\") first to see the current content.", path), nil
	}

	// ファイルを読み込む
	contentBytes, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Sprintf("Error reading file: %v", err), nil
	}

	oldContent := string(contentBytes)
	var newContent string

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
			return fmt.Sprintf("Error: %v", err), nil
		}

		lines := strings.Split(oldContent, "\n")
		if len(lines) == 0 {
			return "Error: file is empty", nil
		}

		if startLine > len(lines) {
			return fmt.Sprintf("Error: start_line is out of range (start_line=%d, file_lines=%d)", startLine, len(lines)), nil
		}
		if endLine > len(lines) {
			return fmt.Sprintf("Error: end_line is out of range (end_line=%d, file_lines=%d)", endLine, len(lines)), nil
		}

		// 指定レンジを new_str の行で置き換える
		newStrLines := strings.Split(newStr, "\n")
		newLines := make([]string, 0, len(lines)-(endLine-startLine+1)+len(newStrLines))
		newLines = append(newLines, lines[:startLine-1]...)
		newLines = append(newLines, newStrLines...)
		newLines = append(newLines, lines[endLine:]...)
		newContent = strings.Join(newLines, "\n")

		if newStr != "" {
			prefixContent := strings.Join(lines[:startLine-1], "\n")
			suffixContent := strings.Join(lines[endLine:], "\n")
			if strings.Contains(prefixContent, newStr) || strings.Contains(suffixContent, newStr) {
				common.Yellow.Println("⚠️  Warning: new_str already exists outside the target range (possible duplication)")
			}
		}

		// 確認UI - 変更サマリーを明確に表示
		removed := endLine - startLine + 1
		added := len(newStrLines)
		lineDiff := added - removed

		common.Cyan.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		common.Cyan.Printf("🔧 str_replace (line range): %s\n", path)
		common.Cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

		common.Yellow.Println("\n📊 Summary / 変更サマリー:")
		fmt.Printf("   • Range: %d-%d (1-indexed inclusive)\n", startLine, endLine)
		fmt.Printf("   • Remove %d lines / %d行削除\n", removed, removed)
		fmt.Printf("   • Add %d lines / %d行追加\n", added, added)
		if lineDiff > 0 {
			common.Green.Printf("   • Net: +%d lines\n", lineDiff)
		} else if lineDiff < 0 {
			common.Red.Printf("   • Net: %d lines\n", lineDiff)
		} else {
			fmt.Printf("   • Net: 0 lines (same size)\n")
		}

		// Before/After（レンジ部分のみを表示）
		beforeStr := strings.Join(lines[startLine-1:endLine], "\n")
		{
			offset := startLine - 1
			cfg := config.GetGlobalConfig()
			opts := &ui.DiffOptions{
				ContextLines:  cfg.Diff.ContextLines,
				ShowLineNums:  true,
				InlineMode:    true,
				MaxTotalLines: 50,
				LineNumOffset: offset,
			}
			if opts.ContextLines == 0 {
				opts.MaxTotalLines = 0
			}
			ui.ShowColoredDiff(beforeStr, newStr, opts)
		}

		dec := common.ConfirmWithAutoApproveDecision("str_replace", "Apply this replacement? / この置換を適用しますか？")
		switch dec.Action {
		case common.ConfirmYes:
			// continue
		case common.ConfirmComment:
			return fmt.Sprintf(`[COMMENT] User provided feedback for str_replace (line range).

Comment:
%s

Next actions:
- Use read_file to verify the correct range.
- Consider using string-based old_str for more precise matching.

IMPORTANT: Do NOT apply the replacement until the user approves.`, strings.TrimSpace(dec.Comment)), nil
		default:
			common.Yellow.Println("⚠️  User cancelled the replacement")
			return fmt.Sprintf(`[CANCELLED] User cancelled str_replace for %s.

Hint: The replacement was not applied. If you need to make this change:
1. Verify the range with read_file
2. Double-check start_line/end_line are correct (1-indexed inclusive)

Do not retry the same replacement.`, path), nil
		}

		if err := os.WriteFile(absPath, []byte(newContent), 0644); err != nil {
			return fmt.Sprintf("Error writing file: %v", err), nil
		}

		common.Green.Printf("✅ Replaced lines %d-%d in: %s\n", startLine, endLine, path)
		msg := fmt.Sprintf("Successfully replaced lines %d-%d in %s", startLine, endLine, path)
		msg += lsp.GetDiagnosticsSummary(absPath)
		return msg, nil
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
		newContent = strings.Replace(oldContent, oldStr, newStr, 1)
	} else if exactMatch && exactCount > 1 {
		// 完全一致が複数 → 改善エラー（Candidates + snippet + Next actions + IMPORTANT）
		lines := strings.Split(oldContent, "\n")
		cands := findAllOccurrencesLineRanges(oldContent, oldStr, 5)

		var b strings.Builder
		fmt.Fprintf(&b, "Error: old_str appears %d times in %s (must be unique).\n\n", exactCount, path)
		b.WriteString("Candidates (1-indexed line ranges, with +/-2 lines context):\n")

		if len(cands) == 0 {
			b.WriteString("- (could not compute candidates)\n")
		} else {
			for _, c := range cands {
				fmt.Fprintf(&b, "\n- lines %d-%d\n", c.StartLine, c.EndLine)
				snip := buildLineSnippet(lines, c.StartLine, c.EndLine, 2)
				b.WriteString(snip)
			}
		}

		b.WriteString("\nNext actions:\n")
		b.WriteString("1) Use read_file to inspect the target area and expand old_str with more surrounding context (e.g. function signature + block).\n")
		b.WriteString("2) If you intended a line-based edit, use delete_lines / insert_before / insert_after.\n")
		b.WriteString("3) If you intended to replace a specific line range, set old_str to empty and provide start_line/end_line (1-indexed inclusive).\n")
		b.WriteString("4) If you want to replace ALL occurrences, use grep_replace: {\"tool\": \"grep_replace\", \"args\": {\"pattern\": \"...\", \"replacement\": \"...\", \"path\": \"file.go\", \"file_pattern\": \"*.go\"}}\n\n")

		previewLines := common.Min(50, len(lines))
		preview := strings.Join(lines[:previewLines], "\n")
		fmt.Fprintf(&b, "File preview (first %d lines):\n---\n%s\n---\n\n", previewLines, preview)

		b.WriteString("IMPORTANT: Do NOT retry the same str_replace with the same old_str. Make old_str unique first.\n")

		return b.String(), nil
	} else {
		// 完全一致しない → 正規化マッチを試行（従来挙動）
		common.Yellow.Println("⚠️  Exact match failed, trying normalized whitespace matching...")

		found, startIdxNormalized, endIdx := common.FindWithNormalizedWhitespace(oldContent, oldStr)
		if !found {
			return fmt.Sprintf("Error: old_str not found in %s (tried both exact and normalized matching)", path), nil
		}

		actualOldStr := oldContent[startIdxNormalized : endIdx+1]
		newContent = oldContent[:startIdxNormalized] + newStr + oldContent[endIdx+1:]

		common.Yellow.Printf("ℹ️  Matched with normalized whitespace (indentation may differ)\n")
		common.Yellow.Printf("   Actual match in file:\n")
		matchLines := strings.Split(actualOldStr, "\n")
		for i, line := range matchLines {
			if i >= 5 {
				common.Yellow.Printf("   ... (%d more lines)\n", len(matchLines)-5)
				break
			}
			common.Yellow.Printf("   │ %s\n", line)
		}
		fmt.Println()
	}

	// 確認UI - 変更サマリーを明確に表示
	oldStrLines := strings.Split(oldStr, "\n")
	newStrLines := strings.Split(newStr, "\n")
	lineDiff := len(newStrLines) - len(oldStrLines)

	common.Cyan.Println("\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	common.Cyan.Printf("🔧 str_replace: %s\n", path)
	common.Cyan.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	common.Yellow.Println("\n📊 Summary / 変更サマリー:")
	fmt.Printf("   • Remove %d lines / %d行削除\n", len(oldStrLines), len(oldStrLines))
	fmt.Printf("   • Add %d lines / %d行追加\n", len(newStrLines), len(newStrLines))
	if lineDiff > 0 {
		common.Green.Printf("   • Net: +%d lines\n", lineDiff)
	} else if lineDiff < 0 {
		common.Red.Printf("   • Net: %d lines\n", lineDiff)
	} else {
		fmt.Printf("   • Net: 0 lines (same size)\n")
	}

	// 大規模変更警告
	absLineDiff := lineDiff
	if absLineDiff < 0 {
		absLineDiff = -absLineDiff
	}
	if absLineDiff > 100 || len(oldStrLines) > 100 || len(newStrLines) > 100 {
		common.Red.Println("\n🚨 VERY LARGE CHANGE DETECTED!")
		common.Red.Println("   非常に大きな変更が検出されました。")
		common.Yellow.Println("💡 Consider splitting into multiple smaller str_replace calls.")
		common.Yellow.Println("   複数の小さな str_replace に分割することを検討してください。")
	}

	// diff表示用に、old_str の実ファイル開始行（1-indexed）を算出
	startLineForDisplay := 1
	if idx := strings.Index(oldContent, oldStr); idx >= 0 {
		// 完全一致（または偶然一致）できる場合は、その位置を採用
		startLineForDisplay = strings.Count(oldContent[:idx], "\n") + 1
	}

	{
		offset := startLineForDisplay - 1
		cfg := config.GetGlobalConfig()
		opts := &ui.DiffOptions{
			ContextLines:  cfg.Diff.ContextLines,
			ShowLineNums:  true,
			InlineMode:    true,
			MaxTotalLines: 50,
			LineNumOffset: offset,
		}
		if opts.ContextLines == 0 {
			opts.MaxTotalLines = 0
		}
		ui.ShowColoredDiff(oldStr, newStr, opts)
	}
	if newStr != "" && strings.Contains(oldContent, newStr) {
		common.Yellow.Println("⚠️  Warning: new_str already exists in file (possible duplication)")
	}

	dec2 := common.ConfirmWithAutoApproveDecision("str_replace", "Apply this replacement? / この置換を適用しますか？")
	switch dec2.Action {
	case common.ConfirmYes:
		// continue
	case common.ConfirmComment:
		return fmt.Sprintf(`[COMMENT] User provided feedback for str_replace.

Comment:
%s

Next actions:
- Use read_file to confirm the old_str location.
- Consider using line-range mode (start_line/end_line) for block replacement.

IMPORTANT: Do NOT apply the replacement until the user approves.`, strings.TrimSpace(dec2.Comment)), nil
	default:
		common.Yellow.Println("⚠️  User cancelled the replacement")
		return fmt.Sprintf(`[CANCELLED] User cancelled str_replace for %s.

Hint: The replacement was not applied. If you need to make this change:
1. Check if the old_str is correct by using read_file
2. Try a smaller, more specific replacement
3. Ask the user for clarification

Do not retry the same replacement.`, path), nil
	}

	if err := os.WriteFile(absPath, []byte(newContent), 0644); err != nil {
		return fmt.Sprintf("Error writing file: %v", err), nil
	}

	common.Green.Printf("✅ Replaced in: %s\n", path)
	msg := fmt.Sprintf("Successfully replaced text in %s", path)
	msg += lsp.GetDiagnosticsSummary(absPath)
	return msg, nil
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

func buildLineSnippet(lines []string, startLine, endLine, ctx int) string {
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

	var b strings.Builder
	for i := start; i <= end; i++ {
		prefix := "  "
		if i >= startLine && i <= endLine {
			prefix = "* "
		}
		fmt.Fprintf(&b, "%s%4d: %s\n", prefix, i, lines[i-1])
	}
	return b.String()
}
