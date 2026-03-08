package common

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// ShowImprovedDiff は改善された差分表示
// ui.ShowColoredDiff を使用してインライン形式で表示
func ShowImprovedDiff(oldStr, newStr string) {
	ShowImprovedDiffWithOutput(DefaultOutput(), oldStr, newStr)
}

// ShowImprovedDiffWithOutput は出力先を指定して差分を表示する。
func ShowImprovedDiffWithOutput(out Output, oldStr, newStr string) {
	if out.SuppressStdout() {
		return
	}
	cfg := config.GetGlobalConfig()

	opts := &ui.DiffOptions{
		ContextLines:  cfg.Diff.ContextLines,
		ShowLineNums:  true,
		InlineMode:    true,
		MaxTotalLines: cfg.Diff.MaxTotalLines,
	}

	ui.ShowColoredDiffToWriter(out.StdoutWriter(), oldStr, newStr, opts)
}

// ShowDiff は差分を表示
func ShowDiff(old, new, filename string) {
	ShowDiffWithOutput(DefaultOutput(), old, new, filename)
}

// ShowDiffWithOutput は出力先を指定して差分を表示する。
func ShowDiffWithOutput(out Output, old, new, filename string) {
	if out.SuppressStdout() {
		return
	}
	out.Yellow.Printf("Changes to: %s\n", filename)
	cfg := config.GetGlobalConfig()

	opts := &ui.DiffOptions{
		ContextLines:  cfg.Diff.ContextLines,
		ShowLineNums:  true,
		InlineMode:    true,
		MaxTotalLines: cfg.Diff.MaxTotalLines,
	}

	ui.ShowColoredDiffToWriter(out.StdoutWriter(), old, new, opts)
}

// ShowPreview は新規ファイルのプレビューを表示
func ShowPreview(content string) {
	ShowPreviewWithOutput(DefaultOutput(), content)
}

// ShowPreviewWithOutput は出力先を指定して新規ファイルのプレビューを表示する。
func ShowPreviewWithOutput(out Output, content string) {
	if out.SuppressStdout() {
		return
	}
	cfg := config.GetGlobalConfig()
	maxLines := cfg.Diff.MaxTotalLines

	out.Println(strings.Repeat("-", 50))
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if maxLines > 0 && i >= maxLines {
			out.Yellow.Printf("... (%d more lines)\n", len(lines)-i)
			break
		}
		out.Printf("%4d: %s\n", i+1, line)
	}
	out.Println(strings.Repeat("-", 50))
}
