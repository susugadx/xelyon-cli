package configscreen

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/susugadx/xelyon-cli/internal/tui/confirmdialog"
	"github.com/susugadx/xelyon-cli/internal/tui/termtext"
	"github.com/susugadx/xelyon-cli/internal/tui/theme"
)

// renderConfigHeader はヘッダー行を構築する。
func (cs *Screen) renderConfigHeader(width int) string {
	title := theme.Config.BgHeader + theme.Config.Bold + theme.Config.FgBright + " Configuration " + theme.Config.Reset
	hints := theme.Config.FgDim + "q:close  s:save  /:filter  r:reset" + theme.Config.Reset
	padding := width - lipgloss.Width("Configuration") - 2 - lipgloss.Width("q:close  s:save  /:filter  r:reset")
	if padding < 2 {
		return termtext.FillANSITextWidth(title, width, theme.Config.BgHeader)
	}
	return termtext.FillANSITextWidth(title+strings.Repeat(" ", padding)+hints, width, theme.Config.BgHeader)
}

// renderConfigStatus はステータスバーを構築する。
func (cs *Screen) renderConfigStatus(width int) string {
	statusMsg := cs.statusText()
	left := " " + configStatusColor(cs.saveStatus) + statusMsg + theme.Config.Reset
	hint := configStatusHint(cs)
	right := theme.Config.FgDim + hint + theme.Config.Reset + " "
	padding := width - lipgloss.Width(statusMsg) - lipgloss.Width(hint) - 3
	if padding < 1 {
		return termtext.FillANSITextWidth(left, width, "")
	}
	return fitANSITextWidth(left+strings.Repeat(" ", padding)+right, width)
}

func configStatusColor(status configSaveStatus) string {
	switch status {
	case statusModified:
		return theme.Config.FgYellow
	case statusFailed:
		return theme.Config.FgRed
	case statusSaving:
		return theme.Config.FgCyan
	default:
		return theme.Config.FgGreen
	}
}

func configStatusHint(cs *Screen) string {
	switch {
	case cs.filterMode:
		return "Enter:apply  Esc:cancel"
	case cs.editMode == editSelect:
		return "j/k:move  Enter:select  Esc:cancel"
	case cs.editMode == editInput:
		return "Enter:confirm  Esc:cancel"
	case cs.editMode == editSlice:
		if cs.editSliceAdding || cs.editSliceEditing {
			return "Enter:confirm  Esc:cancel"
		}
		if field := cs.selectedField(); field != nil && cs.editingGuidanceFileChoices(field.Path) {
			return "Space:toggle  a:custom  d:delete custom  Esc:done"
		}
		return "a:add  d:delete  Enter:edit  Esc:done"
	case cs.editMode == editStructMap:
		switch {
		case cs.editStructAdding:
			return "Enter:confirm  Esc:cancel"
		case cs.editEntryActive && cs.editEntryFieldEdit == "input":
			return "Enter:confirm  Esc:cancel"
		case cs.editEntryActive && cs.editEntryFieldEdit == "slice":
			if cs.editSliceAdding || cs.editSliceEditing {
				return "Enter:confirm  Esc:cancel"
			}
			return "a:add  d:delete  Enter:edit  Esc:done"
		case cs.editEntryActive:
			return "j/k:move  Enter:edit  Space:toggle  Esc:back"
		default:
			return "j/k:move  Enter:edit entry  a:add  d:delete  Esc:done"
		}
	default:
		return "j/k:move  h/l:pane  Enter:edit  Space:toggle"
	}
}

// renderConfigConfirmDialog は終了確認ダイアログを構築する。
func (cs *Screen) renderConfigConfirmDialog(width, height int) string {
	options := []string{"Save and quit", "Discard and quit", "Cancel"}
	return confirmdialog.Render(width, height, "Unsaved changes", options, cs.confirmIdx, theme.Config)
}

func fitANSITextWidth(line string, width int) string {
	return termtext.FillANSITextWidth(line, width, "")
}
