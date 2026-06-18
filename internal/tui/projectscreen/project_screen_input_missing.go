package projectscreen

import tea "github.com/charmbracelet/bubbletea"

func (ps *Screen) handleMissingKey(msg tea.KeyMsg) Command {
	switch {
	case msg.Type == tea.KeyEsc || msg.String() == "q":
		return CommandClose
	case isEnterKey(msg):
		if ps.saveStatus == projectStatusSaving {
			return CommandNone
		}
		ps.saveStatus = projectStatusSaving
		ps.saveError = ""
		ps.message = "creating template"
		return CommandCreateTemplate
	default:
		return CommandNone
	}
}
