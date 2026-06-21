package configscreen

import tea "github.com/charmbracelet/bubbletea"

func (cs *Screen) handleStructMapEdit(msg tea.KeyMsg) (configCommand, tea.Cmd) {
	if cs.editEntryActive {
		return cs.handleStructEntryEdit(msg)
	}
	if cs.editStructAdding {
		return cs.handleStructMapAddKey(msg)
	}
	return cs.handleStructMapListEdit(msg)
}
