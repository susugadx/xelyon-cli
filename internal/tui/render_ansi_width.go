package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

const visualTabWidth = 4

// runeWidth は文字の表示幅を返す。
// lipgloss.Width と同一基準を使い、カーソル位置と描画幅のずれを防ぐ。
func runeWidth(r rune) int {
	if r == '\t' {
		return visualTabWidth
	}
	return lipgloss.Width(string(r))
}

func plainTextDisplayWidth(s string) int {
	return lipgloss.Width(strings.ReplaceAll(s, "\t", strings.Repeat(" ", visualTabWidth)))
}
