package tui

import tea "github.com/charmbracelet/bubbletea"

func (cs *configScreen) handleStructMapListEdit(msg tea.KeyMsg) (configCommand, tea.Cmd) {
	s := msg.String()
	switch {
	case msg.Type == tea.KeyEsc:
		cs.editMode = editNone
		return configCommandNone, nil

	case msg.Type == tea.KeyUp || s == "k":
		if cs.editStructIndex > 0 {
			cs.editStructIndex--
		}
		return configCommandNone, nil

	case msg.Type == tea.KeyDown || s == "j":
		if cs.editStructIndex < len(cs.editStructKeys)-1 {
			cs.editStructIndex++
		}
		return configCommandNone, nil

	case s == "d":
		cs.deleteSelectedStructMapKey()
		return configCommandNone, nil

	case s == "a":
		cs.startStructMapAddKey()
		return configCommandNone, nil

	case isEnterKey(msg):
		cs.openSelectedStructMapEntry()
		return configCommandNone, nil
	}
	return configCommandNone, nil
}

func (cs *configScreen) deleteSelectedStructMapKey() {
	if cs.editStructIndex < 0 || cs.editStructIndex >= len(cs.editStructKeys) {
		return
	}
	field := cs.selectedField()
	if field == nil {
		return
	}
	key := cs.editStructKeys[cs.editStructIndex]
	cs.deleteStructMapKey(field.Path, key)
	cs.editStructKeys = append(cs.editStructKeys[:cs.editStructIndex], cs.editStructKeys[cs.editStructIndex+1:]...)
	if cs.editStructIndex >= len(cs.editStructKeys) && cs.editStructIndex > 0 {
		cs.editStructIndex--
	}
	cs.markModified()
}

func (cs *configScreen) startStructMapAddKey() {
	cs.editStructAdding = true
	cs.editStructInput.SetValue("")
	cs.editStructInput.Focus()
}

func (cs *configScreen) openSelectedStructMapEntry() {
	if cs.editStructIndex < 0 || cs.editStructIndex >= len(cs.editStructKeys) {
		return
	}
	field := cs.selectedField()
	if field == nil {
		return
	}
	key := cs.editStructKeys[cs.editStructIndex]
	fields := cs.loadEntryFields(field.Path, key)
	if len(fields) == 0 {
		return
	}
	cs.editEntryActive = true
	cs.editEntryKey = key
	cs.editEntryFields = fields
	cs.editEntryIndex = 0
	cs.editEntryFieldEdit = ""
}
