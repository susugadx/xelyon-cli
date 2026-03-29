package ui

import (
	"fmt"
	"io"
	"strings"
)

type diffPrinter struct {
	out io.Writer
	pal fileOpPalette
}

func newDiffPrinter(out io.Writer) diffPrinter {
	if out == nil {
		out = io.Discard
	}
	return diffPrinter{out: out, pal: newFileOpPalette(out)}
}

func (p diffPrinter) println(args ...interface{}) {
	_, _ = fmt.Fprintln(p.out, args...)
}

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

// ShowColoredDiffWithRuntime は UI runtime の出力先へ色付きの差分を表示する。
func ShowColoredDiffWithRuntime(runtime *Runtime, oldStr, newStr string, opts *DiffOptions) {
	ShowColoredDiffToWriter(runtimeOrDefault(runtime).Output(), oldStr, newStr, opts)
}

// ShowColoredDiffToWriter は指定 writer へ色付きの差分を表示する。
func ShowColoredDiffToWriter(out io.Writer, oldStr, newStr string, opts *DiffOptions) {
	if opts == nil {
		opts = &DefaultDiffOptions
	}

	p := newDiffPrinter(out)
	oldLines := strings.Split(oldStr, "\n")
	newLines := strings.Split(newStr, "\n")

	p.println()
	fileOpDividerInternal(p.out, p.pal, 50)

	added, removed := CountDiffLines(oldLines, newLines)
	fileOpStatsInternal(p.out, p.pal, removed, added)
	p.println()

	if opts.InlineMode {
		showInlineDiffToWriter(p, oldLines, newLines, opts)
	} else {
		showSideBySideDiffToWriter(p, oldLines, newLines, opts)
	}

	fileOpDividerInternal(p.out, p.pal, 50)
	p.println()
}

// showInlineDiffToWriter はインライン形式で差分を表示する。
func showInlineDiffToWriter(p diffPrinter, oldLines, newLines []string, opts *DiffOptions) {
	type diffLine struct {
		typ     string
		lineNum int
		text    string
	}

	var diffs []diffLine
	i, j := 0, 0

	for i < len(oldLines) || j < len(newLines) {
		if i >= len(oldLines) && j >= len(newLines) {
			break
		}
		if i >= len(oldLines) {
			diffs = append(diffs, diffLine{"+", j + 1, newLines[j]})
			j++
			continue
		}
		if j >= len(newLines) {
			diffs = append(diffs, diffLine{"-", i + 1, oldLines[i]})
			i++
			continue
		}
		if oldLines[i] == newLines[j] {
			diffs = append(diffs, diffLine{"=", i + 1, oldLines[i]})
			i++
			j++
			continue
		}

		foundOld, foundNew := -1, -1
		for look := 1; look < 10; look++ {
			if i+look < len(oldLines) && oldLines[i+look] == newLines[j] {
				foundOld = i + look
				break
			}
			if j+look < len(newLines) && newLines[j+look] == oldLines[i] {
				foundNew = j + look
				break
			}
		}

		if foundOld >= 0 {
			for k := i; k < foundOld; k++ {
				diffs = append(diffs, diffLine{"-", k + 1, oldLines[k]})
			}
			i = foundOld
		} else if foundNew >= 0 {
			for k := j; k < foundNew; k++ {
				diffs = append(diffs, diffLine{"+", k + 1, newLines[k]})
			}
			j = foundNew
		} else {
			diffs = append(diffs, diffLine{"-", i + 1, oldLines[i]})
			diffs = append(diffs, diffLine{"+", j + 1, newLines[j]})
			i++
			j++
		}
	}

	displayedLines := 0
	lastDisplayed := -1

	for idx, d := range diffs {
		isChange := d.typ != "="
		shouldDisplay := isChange
		if !isChange && opts.ContextLines > 0 {
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

		if lastDisplayed >= 0 && idx-lastDisplayed > 1 {
			p.pal.Muted(p.out, "   ...\n")
		}
		lastDisplayed = idx

		if opts.MaxTotalLines > 0 && displayedLines >= opts.MaxTotalLines {
			p.pal.Muted(p.out, fmt.Sprintf("   ... (%d more lines)\n", len(diffs)-idx))
			break
		}

		lineNumStr := ""
		if opts.ShowLineNums {
			lineNum := d.lineNum + opts.LineNumOffset
			lineNumStr = fmt.Sprintf("L%-4d ", lineNum)
		}

		switch d.typ {
		case "-":
			p.pal.DelLine(p.out, fmt.Sprintf("   %s- %s\n", lineNumStr, d.text))
		case "+":
			p.pal.AddLine(p.out, fmt.Sprintf("   %s+ %s\n", lineNumStr, d.text))
		case "=":
			p.pal.Context(p.out, fmt.Sprintf("   %s  %s\n", lineNumStr, d.text))
		}
		displayedLines++
	}
}

// showSideBySideDiffToWriter は左右並列形式で差分を表示する。
func showSideBySideDiffToWriter(p diffPrinter, oldLines, newLines []string, opts *DiffOptions) {
	p.pal.Accent(p.out, "Before / 変更前:")
	p.println()
	p.pal.Border(p.out, "┌"+strings.Repeat("─", 58)+"┐")
	p.println()
	for i, line := range oldLines {
		if opts.MaxTotalLines > 0 && i >= opts.MaxTotalLines/2 {
			p.pal.Border(p.out, "│ ")
			p.pal.Muted(p.out, fmt.Sprintf("... (%d lines omitted)", len(oldLines)-i))
			p.pal.Border(p.out, "\n")
			break
		}
		lineNumStr := ""
		if opts.ShowLineNums {
			lineNum := (i + 1) + opts.LineNumOffset
			lineNumStr = fmt.Sprintf("L%-4d ", lineNum)
		}
		text := truncateLine(line, 50)
		p.pal.Border(p.out, "│ ")
		p.pal.Context(p.out, fmt.Sprintf("%s%-50s", lineNumStr, text))
		p.pal.Border(p.out, " │\n")
	}
	p.pal.Border(p.out, "└"+strings.Repeat("─", 58)+"┘")
	p.println()

	p.println()
	p.pal.Accent(p.out, "After / 変更後:")
	p.println()
	p.pal.Border(p.out, "┌"+strings.Repeat("─", 58)+"┐")
	p.println()
	for i, line := range newLines {
		if opts.MaxTotalLines > 0 && i >= opts.MaxTotalLines/2 {
			p.pal.Border(p.out, "│ ")
			p.pal.Muted(p.out, fmt.Sprintf("... (%d lines omitted)", len(newLines)-i))
			p.pal.Border(p.out, "\n")
			break
		}
		lineNumStr := ""
		if opts.ShowLineNums {
			lineNum := (i + 1) + opts.LineNumOffset
			lineNumStr = fmt.Sprintf("L%-4d ", lineNum)
		}
		text := truncateLine(line, 50)
		p.pal.Border(p.out, "│ ")
		p.pal.Context(p.out, fmt.Sprintf("%s%-50s", lineNumStr, text))
		p.pal.Border(p.out, " │\n")
	}
	p.pal.Border(p.out, "└"+strings.Repeat("─", 58)+"┘")
	p.println()
}

// truncateLine は行を指定幅で切り詰める。
func truncateLine(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-3]) + "..."
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ShowUnifiedDiffWithRuntime は UI runtime の出力先へ Unified Diff を表示する。
func ShowUnifiedDiffWithRuntime(runtime *Runtime, diffOutput string) {
	ShowUnifiedDiffToWriter(runtimeOrDefault(runtime).Output(), diffOutput)
}

// ShowPatchToWriter は指定 writer へ apply_patch 形式のパッチを行番号なしの色付きテキストで表示する。
func ShowPatchToWriter(out io.Writer, patchOutput string) {
	p := newDiffPrinter(out)
	lines := strings.Split(patchOutput, "\n")

	for _, line := range lines {
		if line == "" {
			p.println()
			continue
		}

		switch {
		case strings.HasPrefix(line, "*** Add File:"), strings.HasPrefix(line, "*** Update File:"),
			strings.HasPrefix(line, "*** Delete File:"), strings.HasPrefix(line, "*** Move to:"),
			strings.HasPrefix(line, "*** Begin"), strings.HasPrefix(line, "*** End"):
			p.pal.Accent(p.out, line+"\n")
		case strings.HasPrefix(line, "@@"):
			p.pal.Hunk(p.out, line+"\n")
		case strings.HasPrefix(line, "-"):
			p.pal.DelLine(p.out, line+"\n")
		case strings.HasPrefix(line, "+"):
			p.pal.AddLine(p.out, line+"\n")
		case strings.HasPrefix(line, " "):
			p.pal.Context(p.out, line+"\n")
		default:
			p.println(line)
		}
	}
}

// ShowUnifiedDiffToWriter は指定 writer へ Unified Diff を表示する。
func ShowUnifiedDiffToWriter(out io.Writer, diffOutput string) {
	p := newDiffPrinter(out)
	lines := strings.Split(diffOutput, "\n")

	for _, line := range lines {
		if line == "" {
			p.println()
			continue
		}

		switch {
		case strings.HasPrefix(line, "---"), strings.HasPrefix(line, "+++"), strings.HasPrefix(line, "@@"):
			p.pal.Hunk(p.out, line+"\n")
		case strings.HasPrefix(line, "-"):
			p.pal.DelLine(p.out, line+"\n")
		case strings.HasPrefix(line, "+"):
			p.pal.AddLine(p.out, line+"\n")
		default:
			p.println(line)
		}
	}
}

// fileOpDividerInternal はパッケージ内部用の区切り線描画。
func fileOpDividerInternal(w io.Writer, pal fileOpPalette, width int) {
	pal.Border(w, strings.Repeat("─", width))
	_, _ = fmt.Fprintln(w)
}

// fileOpStatsInternal はパッケージ内部用の変更統計描画。
func fileOpStatsInternal(w io.Writer, pal fileOpPalette, removed, added int) {
	_, _ = fmt.Fprint(w, "  ")
	pal.DelLine(w, fmt.Sprintf("-%d", removed))
	_, _ = fmt.Fprint(w, " / ")
	pal.AddLine(w, fmt.Sprintf("+%d", added))
	net := added - removed
	switch {
	case net > 0:
		_, _ = fmt.Fprint(w, " (net ")
		pal.AddLine(w, fmt.Sprintf("+%d", net))
		_, _ = fmt.Fprintln(w, ")")
	case net < 0:
		_, _ = fmt.Fprint(w, " (net ")
		pal.DelLine(w, fmt.Sprintf("%d", net))
		_, _ = fmt.Fprintln(w, ")")
	default:
		_, _ = fmt.Fprintln(w, " (net 0)")
	}
}
