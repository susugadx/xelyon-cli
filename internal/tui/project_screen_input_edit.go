package tui

import tea "github.com/charmbracelet/bubbletea"

func (ps *projectScreen) handleEditKey(msg tea.KeyMsg) (projectCommand, tea.Cmd) {
	switch ps.editMode {
	case projectEditContext:
		return ps.handleContextEditKey(msg)
	case projectEditLine:
		return ps.handleLineEditKey(msg)
	default:
		return projectCommandNone, nil
	}
}

func (ps *projectScreen) handleContextEditKey(msg tea.KeyMsg) (projectCommand, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEsc:
		ps.cancelEdit()
		return projectCommandNone, nil
	case tea.KeyCtrlS:
		ps.applyContextEdit()
		return projectCommandNone, nil
	default:
		var cmd tea.Cmd
		ps.contextArea, cmd = ps.contextArea.Update(msg)
		return projectCommandNone, cmd
	}
}

func (ps *projectScreen) handleLineEditKey(msg tea.KeyMsg) (projectCommand, tea.Cmd) {
	switch {
	case msg.Type == tea.KeyEsc:
		ps.cancelEdit()
		return projectCommandNone, nil
	case isEnterKey(msg):
		ps.applyLineEdit()
		return projectCommandNone, nil
	default:
		var cmd tea.Cmd
		ps.editInput, cmd = ps.editInput.Update(msg)
		return projectCommandNone, cmd
	}
}
