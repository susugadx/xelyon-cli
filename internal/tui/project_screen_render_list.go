package tui

import (
	"github.com/susugadx/xelyon-cli/internal/tui/termtext"
	"github.com/susugadx/xelyon-cli/internal/tui/theme"
)

func (m Model) renderProjectListLines(width, height int) []string {
	ps := m.projectScreen
	items := ps.selectedItems()
	if len(items) == 0 {
		return []string{
			theme.Config.BgNormal + theme.Config.FgDim + " (empty)" + theme.Config.Reset,
			theme.Config.BgNormal + theme.Config.FgDim + " a:add" + theme.Config.Reset,
		}
	}

	if height <= 0 {
		return nil
	}
	includeHint := height > 1
	itemRows := height
	if includeHint {
		itemRows--
	}
	selectedIndex := ps.selectedItemIndex()
	start := projectListWindowStart(selectedIndex, len(items), itemRows)
	end := min(len(items), start+itemRows)

	lines := make([]string, 0, end-start+1)
	for i := start; i < end; i++ {
		selected := ps.activePane == projectPaneItem && i == ps.selectedItemIndex()
		lines = append(lines, projectListItemLine(items[i], width, selected, ps.activePane == projectPaneItem))
	}
	if includeHint {
		lines = append(lines, theme.Config.BgNormal+theme.Config.FgDim+" a:add  d:delete  Enter:edit"+theme.Config.Reset)
	}
	return lines
}

func projectListItemLine(item string, width int, selected, active bool) string {
	bg, fg := configPaneColors(selected, active)
	prefix := "  "
	if selected {
		prefix = "> "
	}
	line := bg + fg + prefix + projectListItemText(item, width-3) + theme.Config.Reset
	return termtext.FillANSITextWidth(line, width, bg)
}

func projectListItemText(item string, width int) string {
	return termtext.TruncateWithANSI(termtext.SanitizeSingleLineANSI(item), max(0, width))
}

func projectListWindowStart(selectedIndex, itemCount, visibleRows int) int {
	if visibleRows <= 0 || itemCount <= visibleRows {
		return 0
	}
	if selectedIndex < 0 {
		selectedIndex = 0
	}
	if selectedIndex >= itemCount {
		selectedIndex = itemCount - 1
	}
	start := selectedIndex - visibleRows + 1
	if start < 0 {
		return 0
	}
	maxStart := itemCount - visibleRows
	if start > maxStart {
		return maxStart
	}
	return start
}
