package projectscreen

import tea "github.com/charmbracelet/bubbletea"

func (ps *Screen) handleEditKey(msg tea.KeyMsg) (Command, tea.Cmd) {
	switch ps.editMode {
	case projectEditContext:
		return ps.handleContextEditKey(msg)
	case projectEditLine:
		return ps.handleLineEditKey(msg)
	default:
		return CommandNone, nil
	}
}

func (ps *Screen) handleContextEditKey(msg tea.KeyMsg) (Command, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		ps.cancelEdit()
		return CommandNone, nil
	case tea.KeyCtrlS:
		ps.applyContextEdit()
		return CommandNone, nil
	default:
		var cmd tea.Cmd
		ps.contextArea, cmd = ps.contextArea.Update(msg)
		return CommandNone, cmd
	}
}

func (ps *Screen) handleLineEditKey(msg tea.KeyMsg) (Command, tea.Cmd) {
	switch {
	case msg.Type == tea.KeyEsc:
		ps.cancelEdit()
		return CommandNone, nil
	case isEnterKey(msg):
		ps.applyLineEdit()
		return CommandNone, nil
	default:
		var cmd tea.Cmd
		ps.editInput, cmd = ps.editInput.Update(msg)
		return CommandNone, cmd
	}
}
