package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/fatih/color"
)

type diffPrinter struct {
	out io.Writer
}

func newDiffPrinter(out io.Writer) diffPrinter {
	if out == nil {
		out = io.Discard
	}
	return diffPrinter{out: out}
}

func (p diffPrinter) printf(format string, args ...interface{}) {
	_, _ = fmt.Fprintf(p.out, format, args...)
}

func (p diffPrinter) println(args ...interface{}) {
	_, _ = fmt.Fprintln(p.out, args...)
}

func (p diffPrinter) colorPrintf(attrs []color.Attribute, format string, args ...interface{}) {
	_, _ = color.New(attrs...).Fprintf(p.out, format, args...)
}

func (p diffPrinter) colorPrintln(attrs []color.Attribute, args ...interface{}) {
	_, _ = color.New(attrs...).Fprintln(p.out, args...)
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

	p.colorPrintln([]color.Attribute{color.FgCyan}, "\n"+strings.Repeat("─", 62))
	p.colorPrintln([]color.Attribute{color.FgCyan}, "📊 Diff / 差分表示")
	p.colorPrintln([]color.Attribute{color.FgCyan}, strings.Repeat("─", 62))

	removed := len(oldLines)
	added := len(newLines)
	diff := added - removed
	p.printf("   ")
	p.colorPrintf([]color.Attribute{color.FgRed}, "-%d", removed)
	p.printf(" / ")
	p.colorPrintf([]color.Attribute{color.FgGreen}, "+%d", added)
	if diff > 0 {
		p.printf(" (net: ")
		p.colorPrintf([]color.Attribute{color.FgGreen}, "+%d", diff)
		p.printf(")\n")
	} else if diff < 0 {
		p.printf(" (net: ")
		p.colorPrintf([]color.Attribute{color.FgRed}, "%d", diff)
		p.printf(")\n")
	} else {
		p.printf(" (net: 0)\n")
	}
	p.println()

	if opts.InlineMode {
		showInlineDiffToWriter(p, oldLines, newLines, opts)
	} else {
		showSideBySideDiffToWriter(p, oldLines, newLines, opts)
	}

	p.colorPrintln([]color.Attribute{color.FgCyan}, strings.Repeat("─", 62)+"\n")
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
			p.colorPrintln([]color.Attribute{color.FgYellow}, "   ...")
		}
		lastDisplayed = idx

		if opts.MaxTotalLines > 0 && displayedLines >= opts.MaxTotalLines {
			p.colorPrintf([]color.Attribute{color.FgYellow}, "   ... (%d more lines / さらに%d行)\n", len(diffs)-idx, len(diffs)-idx)
			break
		}

		lineNumStr := ""
		if opts.ShowLineNums {
			lineNum := d.lineNum + opts.LineNumOffset
			lineNumStr = fmt.Sprintf("L%-4d ", lineNum)
		}

		switch d.typ {
		case "-":
			p.colorPrintf([]color.Attribute{color.FgRed}, "   %s- %s\n", lineNumStr, d.text)
		case "+":
			p.colorPrintf([]color.Attribute{color.FgGreen}, "   %s+ %s\n", lineNumStr, d.text)
		case "=":
			p.printf("   %s  %s\n", lineNumStr, d.text)
		}
		displayedLines++
	}
}

// showSideBySideDiffToWriter は左右並列形式で差分を表示する。
func showSideBySideDiffToWriter(p diffPrinter, oldLines, newLines []string, opts *DiffOptions) {
	p.colorPrintln([]color.Attribute{color.FgCyan}, "Before / 変更前:")
	p.colorPrintln([]color.Attribute{color.FgCyan}, "┌"+strings.Repeat("─", 58)+"┐")
	for i, line := range oldLines {
		if opts.MaxTotalLines > 0 && i >= opts.MaxTotalLines/2 {
			p.colorPrintf([]color.Attribute{color.FgYellow}, "│ ... (%d lines omitted / 行省略)\n", len(oldLines)-i)
			break
		}
		lineNumStr := ""
		if opts.ShowLineNums {
			lineNum := (i + 1) + opts.LineNumOffset
			lineNumStr = fmt.Sprintf("L%-4d ", lineNum)
		}
		text := truncateLine(line, 50)
		p.colorPrintf([]color.Attribute{color.FgCyan}, "│ %s%-50s │\n", lineNumStr, text)
	}
	p.colorPrintln([]color.Attribute{color.FgCyan}, "└"+strings.Repeat("─", 58)+"┘")

	p.colorPrintln([]color.Attribute{color.FgCyan}, "\nAfter / 変更後:")
	p.colorPrintln([]color.Attribute{color.FgCyan}, "┌"+strings.Repeat("─", 58)+"┐")
	for i, line := range newLines {
		if opts.MaxTotalLines > 0 && i >= opts.MaxTotalLines/2 {
			p.colorPrintf([]color.Attribute{color.FgYellow}, "│ ... (%d lines omitted / 行省略)\n", len(newLines)-i)
			break
		}
		lineNumStr := ""
		if opts.ShowLineNums {
			lineNum := (i + 1) + opts.LineNumOffset
			lineNumStr = fmt.Sprintf("L%-4d ", lineNum)
		}
		text := truncateLine(line, 50)
		p.colorPrintf([]color.Attribute{color.FgCyan}, "│ %s%-50s │\n", lineNumStr, text)
	}
	p.colorPrintln([]color.Attribute{color.FgCyan}, "└"+strings.Repeat("─", 58)+"┘")
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

// ShowPatchToWriter は指定 writer へ apply_patch 形式のパッチを行番号付きで表示する。
func ShowPatchToWriter(out io.Writer, patchOutput string) {
	p := newDiffPrinter(out)
	lines := strings.Split(patchOutput, "\n")

	dim := color.New(color.Faint)
	var oldLine, newLine int // ファイル内行番号（0 = 未設定）

	for _, line := range lines {
		if line == "" {
			p.println()
			continue
		}

		switch {
		case strings.HasPrefix(line, "*** Add File:"), strings.HasPrefix(line, "*** Update File:"),
			strings.HasPrefix(line, "*** Delete File:"), strings.HasPrefix(line, "*** Move to:"),
			strings.HasPrefix(line, "*** Begin"), strings.HasPrefix(line, "*** End"):
			oldLine, newLine = 0, 0
			p.colorPrintln([]color.Attribute{color.Bold, color.FgCyan}, line)
		case strings.HasPrefix(line, "@@"):
			oldLine, newLine = 0, 0
			p.colorPrintln([]color.Attribute{color.FgCyan}, line)
		case strings.HasPrefix(line, "-"):
			oldLine++
			prefix := dim.Sprintf("  %4d      │ ", oldLine)
			fmt.Fprint(out, prefix)
			p.colorPrintln([]color.Attribute{color.FgRed}, line)
		case strings.HasPrefix(line, "+"):
			newLine++
			prefix := dim.Sprintf("       %4d │ ", newLine)
			fmt.Fprint(out, prefix)
			p.colorPrintln([]color.Attribute{color.FgGreen}, line)
		default:
			oldLine++
			newLine++
			prefix := dim.Sprintf("  %4d %4d │ ", oldLine, newLine)
			fmt.Fprint(out, prefix)
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
			p.colorPrintln([]color.Attribute{color.FgCyan}, line)
		case strings.HasPrefix(line, "-"):
			p.colorPrintln([]color.Attribute{color.FgRed}, line)
		case strings.HasPrefix(line, "+"):
			p.colorPrintln([]color.Attribute{color.FgGreen}, line)
		default:
			p.println(line)
		}
	}
}
