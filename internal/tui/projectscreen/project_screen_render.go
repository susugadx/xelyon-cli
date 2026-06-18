package projectscreen

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/susugadx/xelyon-cli/internal/tui/confirmdialog"
	"github.com/susugadx/xelyon-cli/internal/tui/termtext"
	"github.com/susugadx/xelyon-cli/internal/tui/theme"
)

type projectBodyColumn struct {
	width int
	lines []string
}

// View は /project 画面を描画する。
func (ps *Screen) View(width, height int) string {
	if ps == nil {
		return "Loading..."
	}
	if ps.confirmQuit {
		options := []string{"Save and quit", "Discard and quit", "Cancel"}
		return confirmdialog.Render(width, height, "Unsaved project changes", options, ps.confirmIdx, theme.Config)
	}

	bodyHeight := max(1, height-2)
	header := ps.renderProjectHeader(width)
	body := ps.renderProjectBody(width, bodyHeight)
	status := ps.renderProjectStatus(width)
	return header + "\n" + body + "\n" + status
}

func (ps *Screen) renderProjectHeader(width int) string {
	titleText := "Project"
	title := theme.Config.BgHeader + theme.Config.Bold + theme.Config.FgBright + " " + titleText + " " + theme.Config.Reset
	hintText := "q:close  s:save  Enter:edit"
	if ps != nil && ps.missing {
		hintText = "Enter:create  Esc:back"
	}
	hints := theme.Config.FgDim + hintText + theme.Config.Reset
	padding := width - lipgloss.Width(titleText) - 2 - lipgloss.Width(hintText)
	if padding < 2 {
		return termtext.FillANSITextWidth(title, width, theme.Config.BgHeader)
	}
	return termtext.FillANSITextWidth(title+strings.Repeat(" ", padding)+hints, width, theme.Config.BgHeader)
}

func (ps *Screen) renderProjectBody(width, height int) string {
	var columns []projectBodyColumn
	if ps.missing || width < 62 || ps.editMode != projectEditNone {
		columns = []projectBodyColumn{{width: width, lines: ps.renderProjectDetailPane(width, height)}}
	} else {
		leftW := min(30, max(24, width/3))
		rightW := max(0, width-leftW)
		columns = []projectBodyColumn{
			{width: leftW, lines: ps.renderProjectSectionPane(leftW, height)},
			{width: rightW, lines: ps.renderProjectDetailPane(rightW, height)},
		}
	}

	var body strings.Builder
	for row := 0; row < height; row++ {
		if row > 0 {
			body.WriteByte('\n')
		}
		for _, column := range columns {
			body.WriteString(projectBodyLine(column, row))
		}
	}
	return body.String()
}

func projectBodyLine(column projectBodyColumn, row int) string {
	if row < len(column.lines) {
		return termtext.FillANSITextWidth(column.lines[row], column.width, theme.Config.BgNormal)
	}
	return termtext.FillANSITextWidth("", column.width, theme.Config.BgNormal)
}
