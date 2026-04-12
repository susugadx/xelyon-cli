package tui

import "strings"

type configBodyColumn struct {
	width int
	lines []string
}

// configView は config screen の View を構築する。
func (m Model) configView() string {
	if m.configScreen == nil {
		return "Loading..."
	}
	if m.configScreen.confirmQuit {
		return m.renderConfigConfirmDialog()
	}

	bodyHeight := max(1, m.height-2)
	header := m.renderConfigHeader(m.width)
	body := m.renderConfigBody(bodyHeight)
	status := m.renderConfigStatus(m.width)

	return header + "\n" + body + "\n" + status
}

func (m Model) renderConfigBody(height int) string {
	columns := m.configBodyColumns(height)

	var body strings.Builder
	for row := 0; row < height; row++ {
		if row > 0 {
			body.WriteByte('\n')
		}
		for _, column := range columns {
			body.WriteString(configBodyLine(column, row))
		}
	}
	return body.String()
}

func (m Model) configBodyColumns(height int) []configBodyColumn {
	cs := m.configScreen
	leftW, midW, rightW := configPaneWidths(m.width)

	switch {
	case rightW > 0:
		return []configBodyColumn{
			{width: leftW, lines: m.renderConfigCategoryPane(leftW, height)},
			{width: midW, lines: m.renderConfigFieldPane(midW, height)},
			{width: rightW, lines: m.renderConfigDetailPane(rightW, height)},
		}
	case midW > 0:
		midLines := m.renderConfigFieldPane(midW, height)
		if cs.editMode != editNone || cs.activePane == paneDetail {
			midLines = m.renderConfigDetailPane(midW, height)
		}
		return []configBodyColumn{
			{width: leftW, lines: m.renderConfigCategoryPane(leftW, height)},
			{width: midW, lines: midLines},
		}
	default:
		lines := m.renderConfigCategoryPane(leftW, height)
		switch {
		case cs.editMode != editNone || cs.activePane == paneDetail:
			lines = m.renderConfigDetailPane(leftW, height)
		case cs.activePane == paneField:
			lines = m.renderConfigFieldPane(leftW, height)
		}
		return []configBodyColumn{{width: leftW, lines: lines}}
	}
}

func configBodyLine(column configBodyColumn, row int) string {
	if row < len(column.lines) {
		return column.lines[row]
	}
	return fillANSITextWidth("", column.width, cfgBgNormal)
}
