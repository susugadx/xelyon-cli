package tui

import (
	"github.com/susugadx/xelyon-cli/internal/tui/slash"
	"github.com/susugadx/xelyon-cli/internal/tui/slashsuggestions"
)

func (m Model) renderSlashSuggestionRows() []string {
	rows := m.visibleSlashSuggestionRenderRows()
	if len(rows) == 0 {
		return nil
	}
	lines := make([]string, 0, len(rows))
	for _, row := range rows {
		lines = append(lines, slashsuggestions.RenderRowString(row, m.width))
	}
	return lines
}

func (m Model) renderSlashSuggestionRow(suggestion slash.Suggestion, selected bool) string {
	return slashsuggestions.RenderRowString(slashsuggestions.NewRenderRow(suggestion, selected), m.width)
}
