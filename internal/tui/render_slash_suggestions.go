package tui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/susugadx/xelyon-cli/internal/tui/slash"
	"github.com/susugadx/xelyon-cli/internal/tui/termtext"
	"github.com/susugadx/xelyon-cli/internal/tui/theme"
)

func (m Model) renderSlashSuggestionRows() []string {
	rows := m.visibleSlashSuggestionRenderRows()
	if len(rows) == 0 {
		return nil
	}
	lines := make([]string, 0, len(rows))
	for _, row := range rows {
		lines = append(lines, m.renderSlashSuggestionRow(row.Suggestion, row.Selected))
	}
	return lines
}

func (m Model) renderSlashSuggestionRow(suggestion slash.Suggestion, selected bool) string {
	chrome := theme.Chrome
	bg := chrome.SuggestionBg
	prefix := "  "
	if selected {
		bg = chrome.SuggestionSelectedBg
		prefix = "› "
	}

	commandWidth := slashSuggestionCommandWidth(m.width)
	descriptionWidth := max(0, m.width-commandWidth-4)
	label := paddedPlainText(suggestion.Label, commandWidth)
	description := termtext.TruncateWithANSI(termtext.SanitizeSingleLineANSI(suggestion.Description), descriptionWidth)
	line := bg + prefix + chrome.SuggestionCommandFg + label + chrome.Reset + bg
	if descriptionWidth > 0 {
		line += "  " + chrome.SuggestionDescFg + description + chrome.Reset + bg
	}
	return termtext.FillANSITextWidth(line+chrome.Reset, m.width, bg)
}

func slashSuggestionCommandWidth(width int) int {
	if width <= 24 {
		return max(8, width-4)
	}
	return min(28, max(14, width/3))
}

func paddedPlainText(text string, width int) string {
	if width <= 0 {
		return ""
	}
	text = termtext.TruncateWithANSI(termtext.SanitizeSingleLineANSI(text), width)
	padding := width - lipgloss.Width(text)
	if padding > 0 {
		text += strings.Repeat(" ", padding)
	}
	return text
}
