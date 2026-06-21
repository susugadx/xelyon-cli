package configscreen

import (
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
)

func (cs *Screen) handleEntryInputEdit(msg tea.KeyMsg) (configCommand, tea.Cmd) {
	switch {
	case msg.Type == tea.KeyEsc:
		cs.editEntryFieldEdit = ""
		cs.editInput.Blur()
		return configCommandNone, nil

	case isEnterKey(msg):
		if !cs.applySelectedStructEntryInput(cs.editInput.Value()) {
			return configCommandNone, nil
		}
		cs.editEntryFieldEdit = ""
		cs.editInput.Blur()
		return configCommandNone, nil

	default:
		var cmd tea.Cmd
		cs.editInput, cmd = cs.editInput.Update(msg)
		return configCommandNone, cmd
	}
}

func (cs *Screen) applySelectedStructEntryInput(raw string) bool {
	if cs.editEntryIndex < 0 || cs.editEntryIndex >= len(cs.editEntryFields) {
		return false
	}
	ef := &cs.editEntryFields[cs.editEntryIndex]
	updated := *ef
	switch ef.Type {
	case "string":
		updated.Value = raw
	case "int":
		v, err := strconv.Atoi(raw)
		if err != nil {
			return false
		}
		updated.Value = v
	default:
		return false
	}
	if !cs.applyEntryFieldAndMark(updated) {
		return false
	}
	*ef = updated
	return true
}
