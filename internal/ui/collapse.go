package ui

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
)

// DefaultMaxVisibleLines はデフォルトの最大表示行数
const DefaultMaxVisibleLines = 5

// CollapseOutput は長い出力を折りたたみ表示する
// maxLines が 0 の場合は設定から取得
func CollapseOutput(output string, maxLines int) string {
	if maxLines == 0 {
		maxLines = GetMaxVisibleLines()
	}

	lines := strings.Split(output, "\n")

	// 行数が maxLines 以下なら折りたたみ不要
	if len(lines) <= maxLines {
		return output
	}

	// 最初の maxLines 行を表示
	visible := lines[:maxLines]
	hidden := len(lines) - maxLines

	// 折りたたみ表示を作成
	result := strings.Join(visible, "\n")
	result += fmt.Sprintf("\n%s... +%d lines%s", colorDim, hidden, colorReset)

	return result
}

// CollapseOutputWithPrefix は出力を折りたたみ、各行にプレフィックスを付ける
// Claude Code 風の表示用
func CollapseOutputWithPrefix(output string, prefix string, maxLines int) string {
	if maxLines == 0 {
		maxLines = GetMaxVisibleLines()
	}

	lines := strings.Split(output, "\n")

	// 空行のみの場合
	if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
		return ""
	}

	var result strings.Builder
	visibleCount := 0

	for i, line := range lines {
		if visibleCount >= maxLines {
			hidden := len(lines) - i
			fmt.Fprintf(&result, "%s%s... +%d lines%s\n", prefix, colorDim, hidden, colorReset)
			break
		}

		result.WriteString(prefix)
		result.WriteString(line)
		result.WriteString("\n")
		visibleCount++
	}

	return strings.TrimSuffix(result.String(), "\n")
}

// GetMaxVisibleLines は設定から最大表示行数を取得
func GetMaxVisibleLines() int {
	return GetMaxVisibleLinesWithConfig(config.GetGlobalConfig())
}

// GetMaxVisibleLinesWithConfig は設定から最大表示行数を取得する。
func GetMaxVisibleLinesWithConfig(cfg *config.Config) int {
	// 環境変数を優先
	if envVal := os.Getenv("XELYON_OUTPUT_MAX_LINES"); envVal != "" {
		if val, err := strconv.Atoi(envVal); err == nil && val > 0 {
			return val
		}
	}

	// 設定ファイルから取得
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	if cfg != nil && cfg.Output.MaxLines > 0 {
		return cfg.Output.MaxLines
	}

	return DefaultMaxVisibleLines
}

// FormatToolOutput はツール出力をフォーマット（折りたたみ付き）
// Claude Code 風の表示:
//
//	⎿  line1
//	   line2
//	   ... +N lines
func FormatToolOutput(output string, maxLines int) string {
	if output == "" {
		return ""
	}

	lines := strings.Split(output, "\n")

	// 空行のみの場合
	if len(lines) == 0 || (len(lines) == 1 && lines[0] == "") {
		return ""
	}

	if maxLines == 0 {
		maxLines = GetMaxVisibleLines()
	}

	var result strings.Builder

	for i, line := range lines {
		if i >= maxLines {
			hidden := len(lines) - i
			fmt.Fprintf(&result, "   %s... +%d lines%s\n", colorDim, hidden, colorReset)
			break
		}

		if i == 0 {
			result.WriteString("⎿  ")
		} else {
			result.WriteString("   ")
		}
		result.WriteString(line)
		result.WriteString("\n")
	}

	return strings.TrimSuffix(result.String(), "\n")
}
