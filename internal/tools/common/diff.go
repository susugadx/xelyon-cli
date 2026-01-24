package common

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// ShowImprovedDiff は改善された差分表示
// ui.ShowColoredDiff を使用してインライン形式で表示
func ShowImprovedDiff(oldStr, newStr string) {
	cfg := config.GetGlobalConfig()

	opts := &ui.DiffOptions{
		ContextLines:  cfg.Diff.ContextLines,
		ShowLineNums:  true,
		InlineMode:    true,
		MaxTotalLines: 50,
	}
	if opts.ContextLines == 0 {
		opts.MaxTotalLines = 0 // 無制限
	}

	ui.ShowColoredDiff(oldStr, newStr, opts)
}

// ShowDiff は差分を表示
func ShowDiff(old, new, filename string) {
	Yellow.Printf("Changes to: %s\n", filename)
	cfg := config.GetGlobalConfig()

	opts := &ui.DiffOptions{
		ContextLines:  cfg.Diff.ContextLines,
		ShowLineNums:  true,
		InlineMode:    true,
		MaxTotalLines: 50,
	}
	if opts.ContextLines == 0 {
		opts.MaxTotalLines = 0
	}

	ui.ShowColoredDiff(old, new, opts)
}

// ShowPreview は新規ファイルのプレビューを表示
func ShowPreview(content string) {
	fmt.Println(strings.Repeat("-", 50))
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if i >= 20 {
			Yellow.Printf("... (%d more lines)\n", len(lines)-20)
			break
		}
		fmt.Println(line)
	}
	fmt.Println(strings.Repeat("-", 50))
}
