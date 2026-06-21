package projectscreen

import tea "github.com/charmbracelet/bubbletea"

// HandleKey は /project 画面のキー入力を処理する。
func (ps *Screen) HandleKey(msg tea.KeyMsg, agentProcessing bool) (Command, tea.Cmd) {
	if msg.Type == tea.KeyCtrlC {
		if agentProcessing {
			return CommandDelegateCtrlC, nil
		}
		if ps.dirty {
			return ps.tryClose(), nil
		}
		return CommandDelegateCtrlC, nil
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
