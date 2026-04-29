package tui

import tea "github.com/charmbracelet/bubbletea"

func (ps *projectScreen) handleKey(msg tea.KeyMsg, agentProcessing bool) (projectCommand, tea.Cmd) {
	if msg.Type == tea.KeyCtrlC {
		if agentProcessing {
			return projectCommandDelegateCtrlC, nil
		}
		if ps.dirty {
			return ps.tryClose(), nil
		}
		return projectCommandDelegateCtrlC, nil
	}
	if ps.confirmQuit {
		return ps.handleConfirmKey(msg), nil
	}
	if ps.missing {
		return ps.handleMissingKey(msg), nil
	}
	if ps.editMode != projectEditNone {
		return ps.handleEditKey(msg)
	}
	return ps.handleBrowseKey(msg), nil
}
