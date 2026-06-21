package common

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/uifileview"
)

func diffOptionsFromConfig(cfg *config.Config) *uifileview.DiffOptions {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	return &uifileview.DiffOptions{
		ContextLines:  cfg.Diff.ContextLines,
		ShowLineNums:  true,
		InlineMode:    true,
		MaxTotalLines: cfg.Diff.MaxTotalLines,
	}
}

// ShowImprovedDiff は改善された差分表示
// uifileview.ShowColoredDiff を使用してインライン形式で表示
func ShowImprovedDiff(oldStr, newStr string) {
	ShowImprovedDiffWithOutput(DefaultOutput(), oldStr, newStr)
}

// ShowImprovedDiffWithOutput は出力先を指定して差分を表示する。
func ShowImprovedDiffWithOutput(out Output, oldStr, newStr string) {
	ShowImprovedDiffWithOutputAndConfig(out, config.DefaultConfig(), oldStr, newStr)
}

// ShowImprovedDiffWithOutputAndConfig は出力先と設定を指定して差分を表示する。
func ShowImprovedDiffWithOutputAndConfig(out Output, cfg *config.Config, oldStr, newStr string) {
	if out.SuppressStdout() {
		return
	}
	uifileview.ShowColoredDiffToWriter(out.StdoutWriter(), oldStr, newStr, diffOptionsFromConfig(cfg))
}

// ShowDiff は差分を表示
func ShowDiff(old, new, filename string) {
	ShowDiffWithOutput(DefaultOutput(), old, new, filename)
}

// ShowDiffWithOutput は出力先を指定して差分を表示する。
func ShowDiffWithOutput(out Output, old, new, filename string) {
	ShowDiffWithOutputAndConfig(out, config.DefaultConfig(), old, new, filename)
}

// ShowDiffWithOutputAndConfig は出力先と設定を指定して差分を表示する。
func ShowDiffWithOutputAndConfig(out Output, cfg *config.Config, old, new, filename string) {
	if out.SuppressStdout() {
		return
	}
	out.Yellow.Printf("Changes to: %s\n", filename)
	uifileview.ShowColoredDiffToWriter(out.StdoutWriter(), old, new, diffOptionsFromConfig(cfg))
}

// ShowPreview は新規ファイルのプレビューを表示
func ShowPreview(content string) {
	ShowPreviewWithOutput(DefaultOutput(), content)
}

// ShowPreviewWithOutput は出力先を指定して新規ファイルのプレビューを表示する。
func ShowPreviewWithOutput(out Output, content string) {
	ShowPreviewWithOutputAndConfig(out, config.DefaultConfig(), content)
}

// ShowPreviewWithOutputAndConfig は出力先と設定を指定して新規ファイルのプレビューを表示する。
func ShowPreviewWithOutputAndConfig(out Output, cfg *config.Config, content string) {
	if out.SuppressStdout() {
		return
	}
	maxLines := 0
	if cfg != nil {
		maxLines = cfg.Diff.MaxTotalLines
	}

	uifileview.FileOpDivider(out.StdoutWriter(), 50)
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		if maxLines > 0 && i >= maxLines {
			out.Yellow.Printf("... (%d more lines)\n", len(lines)-i)
			break
		}
		out.Printf("%4d: %s\n", i+1, line)
	}
	uifileview.FileOpDivider(out.StdoutWriter(), 50)
}
