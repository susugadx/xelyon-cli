package termtext

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// FillANSITextWidth は ANSI escape を保持しつつ表示幅を指定幅に揃える。
func FillANSITextWidth(line string, width int, bg string) string {
	if width <= 0 {
		return ""
	}
	line = TruncateWithANSI(line, width)
	padding := max(0, width-lipgloss.Width(line))
	if bg == "" {
		return line + strings.Repeat(" ", padding)
	}
	line = strings.ReplaceAll(line, "\033[0m", "\033[0m"+bg)
	line = strings.ReplaceAll(line, "\033[m", "\033[m"+bg)
	line = strings.ReplaceAll(line, "\033[49m", "\033[49m"+bg)
	return bg + line + "\033[0m" + bg + strings.Repeat(" ", padding) + "\033[0m"
}
