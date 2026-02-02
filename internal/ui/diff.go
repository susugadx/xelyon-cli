package ui

import (
	"fmt"
	"strings"
)

// DiffOptions は差分表示のオプション
type DiffOptions struct {
	ContextLines  int  // 差分前後の表示行数
	ShowLineNums  bool // 行番号を表示するか
	InlineMode    bool // インラインモード（追加・削除を連続表示）
	MaxTotalLines int  // 最大表示行数（0=無制限）
	LineNumOffset int  // 表示する行番号のオフセット（0なら従来どおり1-indexed）
}

// DefaultDiffOptions はデフォルトの差分表示オプション
var DefaultDiffOptions = DiffOptions{
	ContextLines:  3,
	ShowLineNums:  true,
	InlineMode:    true,
	MaxTotalLines: 50,
}

// ShowColoredDiff は色付きの差分を表示
// oldStr: 変更前のテキスト
// newStr: 変更後のテキスト
// opts: 表示オプション（nilならデフォルト使用）
func ShowColoredDiff(oldStr, newStr string, opts *DiffOptions) {
	if opts == nil {
		opts = &DefaultDiffOptions
	}

	oldLines := strings.Split(oldStr, "\n")
	newLines := strings.Split(newStr, "\n")

	// ヘッダー
	Cyan.Println("\n" + strings.Repeat("─", 62))
	Cyan.Println("📊 Diff / 差分表示")
	Cyan.Println(strings.Repeat("─", 62))

	// サマリー
	removed := len(oldLines)
	added := len(newLines)
	diff := added - removed
	fmt.Printf("   ")
	Red.Printf("-%d", removed)
	fmt.Printf(" / ")
	Green.Printf("+%d", added)
	if diff > 0 {
		fmt.Printf(" (net: ")
		Green.Printf("+%d", diff)
		fmt.Printf(")\n")
	} else if diff < 0 {
		fmt.Printf(" (net: ")
		Red.Printf("%d", diff)
		fmt.Printf(")\n")
	} else {
		fmt.Printf(" (net: 0)\n")
	}
	fmt.Println()

	if opts.InlineMode {
		showInlineDiff(oldLines, newLines, opts)
	} else {
		showSideBySideDiff(oldLines, newLines, opts)
	}

	Cyan.Println(strings.Repeat("─", 62) + "\n")
}

// showInlineDiff はインライン形式で差分を表示
func showInlineDiff(oldLines, newLines []string, opts *DiffOptions) {
	// LCS (Longest Common Subsequence) ベースの簡易差分
	// 行単位で比較して変更箇所を特定

	type diffLine struct {
		typ     string // "=", "-", "+"
		lineNum int    // 行番号（1-indexed）
		text    string
	}

	var diffs []diffLine
	i, j := 0, 0

	for i < len(oldLines) || j < len(newLines) {
		// 両方終了
		if i >= len(oldLines) && j >= len(newLines) {
			break
		}

		// oldが終了 → 残りは追加
		if i >= len(oldLines) {
			diffs = append(diffs, diffLine{"+", j + 1, newLines[j]})
			j++
			continue
		}

		// newが終了 → 残りは削除
		if j >= len(newLines) {
			diffs = append(diffs, diffLine{"-", i + 1, oldLines[i]})
			i++
			continue
		}

		// 同じ行
		if oldLines[i] == newLines[j] {
			diffs = append(diffs, diffLine{"=", i + 1, oldLines[i]})
			i++
			j++
			continue
		}

		// 異なる行 → 次の一致点を探す
		foundOld, foundNew := -1, -1
		for look := 1; look < 10; look++ {
			// oldLines[i+look]がnewLines[j]と一致するか
			if i+look < len(oldLines) && oldLines[i+look] == newLines[j] {
				foundOld = i + look
				break
			}
			// newLines[j+look]がoldLines[i]と一致するか
			if j+look < len(newLines) && newLines[j+look] == oldLines[i] {
				foundNew = j + look
				break
			}
		}

		if foundOld >= 0 {
			// old側に一致があった → それまでは削除
			for k := i; k < foundOld; k++ {
				diffs = append(diffs, diffLine{"-", k + 1, oldLines[k]})
			}
			i = foundOld
		} else if foundNew >= 0 {
			// new側に一致があった → それまでは追加
			for k := j; k < foundNew; k++ {
				diffs = append(diffs, diffLine{"+", k + 1, newLines[k]})
			}
			j = foundNew
		} else {
			// 一致なし → 削除して追加
			diffs = append(diffs, diffLine{"-", i + 1, oldLines[i]})
			diffs = append(diffs, diffLine{"+", j + 1, newLines[j]})
			i++
			j++
		}
	}

	// コンテキストを考慮して表示
	displayedLines := 0
	lastDisplayed := -1

	for idx, d := range diffs {
		// 変更行かどうか
		isChange := d.typ != "="

		// 表示するかの判定
		shouldDisplay := isChange

		// コンテキスト行（変更行の前後）
		if !isChange && opts.ContextLines > 0 {
			// 前後にある変更行を確認
			for k := max(0, idx-opts.ContextLines); k <= min(len(diffs)-1, idx+opts.ContextLines); k++ {
				if diffs[k].typ != "=" {
					shouldDisplay = true
					break
				}
			}
		}

		if !shouldDisplay {
			continue
		}

		// 省略記号（連続していない場合）
		if lastDisplayed >= 0 && idx-lastDisplayed > 1 {
			Yellow.Println("   ...")
		}
		lastDisplayed = idx

		// 最大行数チェック
		if opts.MaxTotalLines > 0 && displayedLines >= opts.MaxTotalLines {
			Yellow.Printf("   ... (%d more lines / さらに%d行)\n", len(diffs)-idx, len(diffs)-idx)
			break
		}

		// 行を表示
		lineNumStr := ""
		if opts.ShowLineNums {
			lineNum := d.lineNum + opts.LineNumOffset
			lineNumStr = fmt.Sprintf("L%-4d ", lineNum)
		}

		switch d.typ {
		case "-":
			Red.Printf("   %s- %s\n", lineNumStr, d.text)
		case "+":
			Green.Printf("   %s+ %s\n", lineNumStr, d.text)
		case "=":
			fmt.Printf("   %s  %s\n", lineNumStr, d.text)
		}
		displayedLines++
	}
}

// showSideBySideDiff は左右並列形式で差分を表示（Before/After）
func showSideBySideDiff(oldLines, newLines []string, opts *DiffOptions) {
	// Before
	Cyan.Println("Before / 変更前:")
	Cyan.Println("┌" + strings.Repeat("─", 58) + "┐")
	for i, line := range oldLines {
		if opts.MaxTotalLines > 0 && i >= opts.MaxTotalLines/2 {
			Yellow.Printf("│ ... (%d lines omitted / 行省略)\n", len(oldLines)-i)
			break
		}
		lineNumStr := ""
		if opts.ShowLineNums {
			lineNum := (i + 1) + opts.LineNumOffset
			lineNumStr = fmt.Sprintf("L%-4d ", lineNum)
		}

		text := truncateLine(line, 50)
		Cyan.Printf("│ %s%-50s │\n", lineNumStr, text)
	}
	Cyan.Println("└" + strings.Repeat("─", 58) + "┘")

	// After
	Cyan.Println("\nAfter / 変更後:")
	Cyan.Println("┌" + strings.Repeat("─", 58) + "┐")
	for i, line := range newLines {
		if opts.MaxTotalLines > 0 && i >= opts.MaxTotalLines/2 {
			Yellow.Printf("│ ... (%d lines omitted / 行省略)\n", len(newLines)-i)
			break
		}
		lineNumStr := ""
		if opts.ShowLineNums {
			lineNum := (i + 1) + opts.LineNumOffset
			lineNumStr = fmt.Sprintf("L%-4d ", lineNum)
		}

		text := truncateLine(line, 50)
		Cyan.Printf("│ %s%-50s │\n", lineNumStr, text)
	}
	Cyan.Println("└" + strings.Repeat("─", 58) + "┘")
}

// truncateLine は行を指定幅で切り詰め
func truncateLine(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-3]) + "..."
}

// max は2つの整数の大きい方を返す
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// min は2つの整数の小さい方を返す
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ShowUnifiedDiff はUnified Diff形式で色付き表示
func ShowUnifiedDiff(diffOutput string) {
	lines := strings.Split(diffOutput, "\n")

	for _, line := range lines {
		if line == "" {
			fmt.Println()
			continue
		}

		switch {
		case strings.HasPrefix(line, "---"):
			Cyan.Println(line)
		case strings.HasPrefix(line, "+++"):
			Cyan.Println(line)
		case strings.HasPrefix(line, "@@"):
			Cyan.Println(line)
		case strings.HasPrefix(line, "-"):
			Red.Println(line)
		case strings.HasPrefix(line, "+"):
			Green.Println(line)
		default:
			fmt.Println(line)
		}
	}
}
