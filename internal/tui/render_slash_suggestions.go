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
		lines = append(lines, m.renderSlashSuggestionRenderRow(row))
	}
	return lines
}

func (m Model) renderSlashSuggestionRow(suggestion slash.Suggestion, selected bool) string {
	return m.renderSlashSuggestionRenderRow(newSlashSuggestionRenderRow(suggestion, selected))
}

type slashSuggestionRowLayout struct {
	commandWidth     int
	descriptionWidth int
}

func (m Model) renderSlashSuggestionRenderRow(row slashSuggestionRenderRow) string {
	chrome := theme.Chrome
	bg := chrome.SuggestionBg
	prefix := "  "
	prefixFg := chrome.SuggestionPrefixFg
	commandFg := chrome.SuggestionCommandFg
	descriptionFg := chrome.SuggestionDescFg
	if row.Selected {
		bg = chrome.SuggestionSelectedBg
		prefix = "› "
		prefixFg = chrome.SuggestionSelectedFg
		commandFg = chrome.SuggestionSelectedFg
		descriptionFg = chrome.SuggestionSelectedDimFg
	}

	layout := slashSuggestionRowLayoutForWidth(m.width)
	label := paddedPlainText(row.CommandLabel, layout.commandWidth)
	description := termtext.TruncateWithANSI(termtext.SanitizeSingleLineANSI(row.Description), layout.descriptionWidth)
	line := bg + prefixFg + prefix
	line += commandFg + label + chrome.Reset + bg
	if layout.descriptionWidth > 0 {
		line += prefixFg + "  " + chrome.Reset + bg + descriptionFg + description + chrome.Reset + bg
	}
	return termtext.FillANSITextWidth(line+chrome.Reset, m.width, bg)
}

func slashSuggestionRowLayoutForWidth(width int) slashSuggestionRowLayout {
	commandWidth := slashSuggestionCommandWidth(width)
	separatorWidth := 4
	return slashSuggestionRowLayout{
		commandWidth:     commandWidth,
		descriptionWidth: max(0, width-commandWidth-separatorWidth),
	}
}

func slashSuggestionCommandWidth(width int) int {
	if width <= 24 {
		return max(8, width-4)
	}
	if width <= 44 {
		return min(24, max(14, width/2))
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
