package common

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// ShowImprovedDiff は改善された差分表示
// ui.ShowColoredDiff を使用してインライン形式で表示
func ShowImprovedDiff(oldStr, newStr string) {
	if IsQuietMode() {
		return
	}
	cfg := config.GetGlobalConfig()

	opts := &ui.DiffOptions{
		ContextLines:  cfg.Diff.ContextLines,
		ShowLineNums:  true,
		InlineMode:    true,
		MaxTotalLines: cfg.Diff.MaxTotalLines,
	}

	ui.ShowColoredDiff(oldStr, newStr, opts)
}

// ShowDiff は差分を表示
func ShowDiff(old, new, filename string) {
	if IsQuietMode() {
		return
	}
	Yellow.Printf("Changes to: %s\n", filename)
	cfg := config.GetGlobalConfig()

	opts := &ui.DiffOptions{
		ContextLines:  cfg.Diff.ContextLines,
		ShowLineNums:  true,
		InlineMode:    true,
		MaxTotalLines: cfg.Diff.MaxTotalLines,
	}

	ui.ShowColoredDiff(old, new, opts)
}

// ShowPreview は新規ファイルのプレビューを表示
func ShowPreview(content string) {
	if IsQuietMode() {
		return
	}
	cfg := config.GetGlobalConfig()
	maxLines := cfg.Diff.MaxTotalLines

	Println(strings.Repeat("-", 50))
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if maxLines > 0 && i >= maxLines {
			Yellow.Printf("... (%d more lines)\n", len(lines)-i)
			break
		}
		Printf("%4d: %s\n", i+1, line)
	}
	Println(strings.Repeat("-", 50))
}
