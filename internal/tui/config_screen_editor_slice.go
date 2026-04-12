package tui

import tea "github.com/charmbracelet/bubbletea"

func (cs *configScreen) handleSliceEdit(msg tea.KeyMsg) (configCommand, tea.Cmd) {
	if cs.editSliceAdding || cs.editSliceEditing {
		return cs.handleSliceInputKey(msg)
	}

	field := cs.selectedField()
	if field == nil {
		cs.editMode = editNone
		return configCommandNone, nil
	}
	return cs.handleFieldSliceListKey(msg, field)
}

func (cs *configScreen) handleEntrySliceEdit(msg tea.KeyMsg) (configCommand, tea.Cmd) {
	if cs.editSliceAdding || cs.editSliceEditing {
		return cs.handleSliceInputKey(msg)
	}
	return cs.handleEntrySliceListKey(msg)
}
