package tui

import tea "github.com/charmbracelet/bubbletea"

func (ps *projectScreen) handleMissingKey(msg tea.KeyMsg) projectCommand {
	switch {
	case msg.Type == tea.KeyEsc || msg.String() == "q":
		return projectCommandClose
	case isEnterKey(msg):
		if ps.saveStatus == projectStatusSaving {
			return projectCommandNone
		}
		return projectCommandCreateTemplate
	default:
		return projectCommandNone
	}
}
