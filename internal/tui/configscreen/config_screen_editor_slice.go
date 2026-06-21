package configscreen

import tea "github.com/charmbracelet/bubbletea"

func (cs *Screen) handleSliceEdit(msg tea.KeyMsg) (configCommand, tea.Cmd) {
	if cs.editSliceAdding || cs.editSliceEditing {
		return cs.handleSliceInputKey(msg)
	}

	field := cs.selectedField()
	if field == nil {
		cs.editMode = editNone
		return configCommandNone, nil
	}
	if cs.editingGuidanceFileChoices(field.Path) {
		return cs.handleGuidanceFileChoiceKey(msg, field)
	}
	return cs.handleFieldSliceListKey(msg, field)
}

func (cs *Screen) handleEntrySliceEdit(msg tea.KeyMsg) (configCommand, tea.Cmd) {
	if cs.editSliceAdding || cs.editSliceEditing {
		return cs.handleSliceInputKey(msg)
	}
	return cs.handleEntrySliceListKey(msg)
}
