package configscreen

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tui/termtext"
	"github.com/susugadx/xelyon-cli/internal/tui/theme"
)

type configBodyColumn struct {
	width int
	lines []string
}

// View は config screen の View を構築する。
func (cs *Screen) View(width, height int) string {
	if cs == nil {
		return "Loading..."
	}
	if cs.confirmQuit {
		return cs.renderConfigConfirmDialog(width, height)
	}

	bodyHeight := max(1, height-2)
	header := cs.renderConfigHeader(width)
	body := cs.renderConfigBody(width, bodyHeight)
	status := cs.renderConfigStatus(width)

	return header + "\n" + body + "\n" + status
}

func (cs *Screen) renderConfigBody(width, height int) string {
	columns := cs.configBodyColumns(width, height)

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

func (cs *Screen) configBodyColumns(width, height int) []configBodyColumn {
	leftW, midW, rightW := PaneWidths(width)

	switch {
	case rightW > 0:
		return []configBodyColumn{
			{width: leftW, lines: cs.renderConfigCategoryPane(leftW, height)},
			{width: midW, lines: cs.renderConfigFieldPane(midW, height)},
			{width: rightW, lines: cs.renderConfigDetailPane(rightW, height)},
		}
	case midW > 0:
		midLines := cs.renderConfigFieldPane(midW, height)
		if cs.editMode != editNone || cs.activePane == paneDetail {
			midLines = cs.renderConfigDetailPane(midW, height)
		}
		return []configBodyColumn{
			{width: leftW, lines: cs.renderConfigCategoryPane(leftW, height)},
			{width: midW, lines: midLines},
		}
	default:
		lines := cs.renderConfigCategoryPane(leftW, height)
		switch {
		case cs.editMode != editNone || cs.activePane == paneDetail:
			lines = cs.renderConfigDetailPane(leftW, height)
		case cs.activePane == paneField:
			lines = cs.renderConfigFieldPane(leftW, height)
		}
		return []configBodyColumn{{width: leftW, lines: lines}}
	}
}

func configBodyLine(column configBodyColumn, row int) string {
	if row < len(column.lines) {
		return column.lines[row]
	}
	return termtext.FillANSITextWidth("", column.width, theme.Config.BgNormal)
}
