package tui

import (
	"github.com/susugadx/xelyon-cli/internal/commandcatalog"
	"github.com/susugadx/xelyon-cli/internal/tui/termtext"
	"github.com/susugadx/xelyon-cli/internal/tui/theme"
)

func (m Model) renderSlashSuggestionRows() []string {
	rows := m.visibleSlashSuggestionRows()
	if len(rows) == 0 {
		return nil
	}
	lines := make([]string, 0, len(rows))
	start := m.slashSuggestionWindowStart()
	for i, cmd := range rows {
		selected := start+i == m.slashSuggestions.selected
		lines = append(lines, m.renderSlashSuggestionRow(cmd, selected))
	}
	return lines
}

func (m Model) renderSlashSuggestionRow(cmd commandcatalog.CommandInfo, selected bool) string {
	chrome := theme.Chrome
	bg := chrome.SuggestionBg
	prefix := "  "
	if selected {
		bg = chrome.SuggestionSelectedBg
		prefix = "› "
	}

	commandWidth := slashSuggestionCommandWidth(m.width)
	descriptionWidth := max(0, m.width-commandWidth-4)
	label := paddedPlainText(slashSuggestionLabel(cmd), commandWidth)
	description := termtext.TruncateWithANSI(termtext.SanitizeSingleLineANSI(cmd.Description), descriptionWidth)
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
