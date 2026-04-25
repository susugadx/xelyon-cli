package termtext

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// VisualTabWidth は TUI 表示上の tab 幅を表す。
const VisualTabWidth = 4

// RuneWidth は文字の表示幅を返す。
// lipgloss.Width と同一基準を使い、カーソル位置と描画幅のずれを防ぐ。
func RuneWidth(r rune) int {
	if r == '\t' {
		return VisualTabWidth
	}
	return lipgloss.Width(string(r))
}

// PlainTextDisplayWidth は ANSI を含まない文字列の表示幅を返す。
func PlainTextDisplayWidth(s string) int {
	return lipgloss.Width(strings.ReplaceAll(s, "\t", strings.Repeat(" ", VisualTabWidth)))
}
