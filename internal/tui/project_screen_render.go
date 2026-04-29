package tui

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

func (m Model) projectView() string {
	if m.projectScreen == nil {
		return "Loading..."
	}
	if m.projectScreen.confirmQuit {
		options := []string{"Save and quit", "Discard and quit", "Cancel"}
		return confirmdialog.Render(m.width, m.height, "Unsaved project changes", options, m.projectScreen.confirmIdx, theme.Config)
	}

	bodyHeight := max(1, m.height-2)
	header := m.renderProjectHeader(m.width)
	body := m.renderProjectBody(bodyHeight)
	status := m.renderProjectStatus(m.width)
	return header + "\n" + body + "\n" + status
}

func (m Model) renderProjectHeader(width int) string {
	titleText := "Project"
	title := theme.Config.BgHeader + theme.Config.Bold + theme.Config.FgBright + " " + titleText + " " + theme.Config.Reset
	hintText := "q:close  s:save  Enter:edit"
	if m.projectScreen != nil && m.projectScreen.missing {
		hintText = "Enter:create  Esc:back"
	}
	hints := theme.Config.FgDim + hintText + theme.Config.Reset
	padding := width - lipgloss.Width(titleText) - 2 - lipgloss.Width(hintText)
	if padding < 2 {
		return termtext.FillANSITextWidth(title, width, theme.Config.BgHeader)
	}
	return termtext.FillANSITextWidth(title+strings.Repeat(" ", padding)+hints, width, theme.Config.BgHeader)
}

func (m Model) renderProjectBody(height int) string {
	ps := m.projectScreen
	var columns []projectBodyColumn
	if ps.missing || m.width < 62 || ps.editMode != projectEditNone {
		columns = []projectBodyColumn{{width: m.width, lines: m.renderProjectDetailPane(m.width, height)}}
	} else {
		leftW := min(30, max(24, m.width/3))
		rightW := max(0, m.width-leftW)
		columns = []projectBodyColumn{
			{width: leftW, lines: m.renderProjectSectionPane(leftW, height)},
			{width: rightW, lines: m.renderProjectDetailPane(rightW, height)},
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
