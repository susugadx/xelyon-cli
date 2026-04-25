package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tui/termtext"
	"github.com/susugadx/xelyon-cli/internal/tui/theme"
)

// renderConfigCategoryPane は左ペイン（カテゴリ一覧）を構築する。
func (m Model) renderConfigCategoryPane(width, height int) []string {
	cs := m.configScreen
	isActive := cs.activePane == paneCategory

	lines := make([]string, 0, height)
	for i, cat := range cs.categories {
		selected := i == cs.catIndex
		bg, fg := configPaneColors(selected, isActive)

		prefix := "  "
		if selected {
			prefix = "> "
		}
		line := bg + fg + prefix + termtext.TruncateWithANSI(cat.DisplayName, width-3) + theme.Config.Reset
		lines = append(lines, termtext.FillANSITextWidth(line, width, bg))
	}

	return appendConfigPanePadding(lines, width, height, theme.Config.BgNormal)
}

// renderConfigFieldPane は中ペイン（フィールド一覧）を構築する。
func (m Model) renderConfigFieldPane(width, height int) []string {
	if width <= 0 {
		return nil
	}

	cs := m.configScreen
	fields := cs.filteredFields()
	hasFilter := cs.filterMode || cs.filterText != ""
	itemRows := visibleConfigFieldRows(height, hasFilter)
	start, end := visibleConfigFieldRange(cs.fieldScroll, len(fields), itemRows)

	lines := make([]string, 0, height)
	for idx := start; idx < end; idx++ {
		lines = append(lines, m.renderConfigFieldRow(fields[idx], idx == cs.fieldIndex, cs.activePane == paneField, width))
	}

	lines = appendConfigPanePadding(lines, width, itemRows, theme.Config.BgNormal)
	if hasFilter {
		lines = append(lines, m.renderConfigFilterRow(width))
	}
	return lines
}

func configPaneColors(selected, active bool) (string, string) {
	switch {
	case selected && active:
		return theme.Config.BgSelected, theme.Config.FgBright
	case selected:
		return theme.Config.BgInactive, theme.Config.FgNormal
	default:
		return theme.Config.BgNormal, theme.Config.FgNormal
	}
}

func appendConfigPanePadding(lines []string, width, height int, bg string) []string {
	for len(lines) < height {
		lines = append(lines, termtext.FillANSITextWidth("", width, bg))
	}
	return lines
}

func visibleConfigFieldRows(height int, hasFilter bool) int {
	itemRows := height
	if hasFilter {
		itemRows--
	}
	if itemRows < 0 {
		return 0
	}
	return itemRows
}

func visibleConfigFieldRange(scroll, total, rows int) (int, int) {
	start := min(scroll, total)
	end := min(start+rows, total)
	return start, end
}

func (m Model) renderConfigFieldRow(field config.ConfigField, selected, active bool, width int) string {
	bg, fg := configPaneColors(selected, active)
	prefix := "  "
	if selected {
		prefix = "> "
	}

	if field.FieldType == config.FieldTypeBool {
		val := "[ ]"
		if b, _ := field.Current.(bool); b {
			val = "[x]"
		}
		line := bg + fg + prefix + termtext.TruncateWithANSI(field.DisplayName+" "+val, width-3) + theme.Config.Reset
		return termtext.FillANSITextWidth(line, width, bg)
	}

	maxNameW := width - 3
	nameW := lipgloss.Width(field.DisplayName) + 1
	valW := max(0, maxNameW-nameW)
	truncVal := termtext.TruncateWithANSI(formatConfigValue(field.Current, field.FieldType), valW)
	line := bg + fg + prefix + termtext.TruncateWithANSI(field.DisplayName, maxNameW) + " " + theme.Config.FgDim + truncVal + theme.Config.Reset
	return termtext.FillANSITextWidth(line, width, bg)
}

func (m Model) renderConfigFilterRow(width int) string {
	cs := m.configScreen
	filterLine := theme.Config.BgHeader + theme.Config.FgCyan + " /" + cs.filterText + theme.Config.Reset
	if cs.filterMode {
		filterLine = theme.Config.BgHeader + theme.Config.FgCyan + " " + cs.filterInput.View() + theme.Config.Reset
	}
	return termtext.FillANSITextWidth(filterLine, width, theme.Config.BgHeader)
}
