package projectscreen

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/susugadx/xelyon-cli/internal/tui/termtext"
	"github.com/susugadx/xelyon-cli/internal/tui/theme"
)

func (ps *Screen) renderProjectStatus(width int) string {
	statusText := projectStatusText(ps)
	left := " " + projectStatusColor(ps.saveStatus) + statusText + theme.Config.Reset
	hint := projectStatusHint(ps)
	right := theme.Config.FgDim + hint + theme.Config.Reset + " "
	padding := width - lipgloss.Width(statusText) - lipgloss.Width(hint) - 3
	if padding < 1 {
		return termtext.FillANSITextWidth(left, width, "")
	}
	return termtext.FillANSITextWidth(left+strings.Repeat(" ", padding)+right, width, "")
}

func projectStatusText(ps *Screen) string {
	if ps == nil {
		return "project"
	}
	if ps.saveError != "" {
		return "project: " + ps.saveError
	}
	if ps.message != "" {
		return "project: " + ps.message
	}
	switch ps.saveStatus {
	case projectStatusModified:
		return "project modified"
	case projectStatusSaving:
		return "project saving"
	case projectStatusFailed:
		return "project save failed"
	default:
		return "project saved"
	}
}

func projectStatusColor(status projectSaveStatus) string {
	switch status {
	case projectStatusModified:
		return theme.Config.FgYellow
	case projectStatusSaving:
		return theme.Config.FgCyan
	case projectStatusFailed:
		return theme.Config.FgRed
	default:
		return theme.Config.FgGreen
	}
}

func projectStatusHint(ps *Screen) string {
	if ps == nil {
		return ""
	}
	switch {
	case ps.missing:
		return "Enter:create  Esc:back"
	case ps.editMode == projectEditContext:
		return "Ctrl+S:confirm  Esc:cancel"
	case ps.editMode == projectEditLine:
		return "Enter:confirm  Esc:cancel"
	case ps.activePane == projectPaneItem:
		return "j/k:move  Enter:edit  a:add  d:delete  Esc:sections"
	default:
		return "j/k:move  l:items  Enter:edit  s:save  q:close"
	}
}
