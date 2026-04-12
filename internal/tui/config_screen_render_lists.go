package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/susugadx/xelyon-cli/internal/config"
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
		line := bg + fg + prefix + truncateWithANSI(cat.DisplayName, width-3) + cfgReset
		lines = append(lines, fillANSITextWidth(line, width, bg))
	}

	return appendConfigPanePadding(lines, width, height, cfgBgNormal)
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

	lines = appendConfigPanePadding(lines, width, itemRows, cfgBgNormal)
	if hasFilter {
		lines = append(lines, m.renderConfigFilterRow(width))
	}
	return lines
}

func configPaneColors(selected, active bool) (string, string) {
	switch {
	case selected && active:
		return cfgBgSelected, cfgFgBright
	case selected:
		return cfgBgInactive, cfgFgNormal
	default:
		return cfgBgNormal, cfgFgNormal
	}
}

func appendConfigPanePadding(lines []string, width, height int, bg string) []string {
	for len(lines) < height {
		lines = append(lines, fillANSITextWidth("", width, bg))
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
		line := bg + fg + prefix + truncateWithANSI(field.DisplayName+" "+val, width-3) + cfgReset
		return fillANSITextWidth(line, width, bg)
	}

	maxNameW := width - 3
	nameW := lipgloss.Width(field.DisplayName) + 1
	valW := max(0, maxNameW-nameW)
	truncVal := truncateWithANSI(formatConfigValue(field.Current, field.FieldType), valW)
	line := bg + fg + prefix + truncateWithANSI(field.DisplayName, maxNameW) + " " + cfgFgDim + truncVal + cfgReset
	return fillANSITextWidth(line, width, bg)
}

func (m Model) renderConfigFilterRow(width int) string {
	cs := m.configScreen
	filterLine := cfgBgHeader + cfgFgCyan + " /" + cs.filterText + cfgReset
	if cs.filterMode {
		filterLine = cfgBgHeader + cfgFgCyan + " " + cs.filterInput.View() + cfgReset
	}
	return fillANSITextWidth(filterLine, width, cfgBgHeader)
}
