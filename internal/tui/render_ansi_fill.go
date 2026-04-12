package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func decorateViewportLine(line string, width int, bg string) string {
	return fillANSITextWidth(line, width, bg)
}

func fitANSITextWidth(line string, width int) string {
	return fillANSITextWidth(line, width, "")
}

func fillANSITextWidth(line string, width int, bg string) string {
	if width <= 0 {
		return ""
	}
	line = truncateWithANSI(line, width)
	padding := max(0, width-lipgloss.Width(line))
	if bg == "" {
		return line + strings.Repeat(" ", padding)
	}
	line = strings.ReplaceAll(line, "\033[0m", "\033[0m"+bg)
	line = strings.ReplaceAll(line, "\033[m", "\033[m"+bg)
	line = strings.ReplaceAll(line, "\033[49m", "\033[49m"+bg)
	return bg + line + "\033[0m" + bg + strings.Repeat(" ", padding) + "\033[0m"
}
