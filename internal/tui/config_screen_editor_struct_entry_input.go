package tui

import (
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
)

func (cs *configScreen) handleEntryInputEdit(msg tea.KeyMsg) (configCommand, tea.Cmd) {
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

func (cs *configScreen) applySelectedStructEntryInput(raw string) bool {
	if cs.editEntryIndex < 0 || cs.editEntryIndex >= len(cs.editEntryFields) {
		return false
	}
	ef := &cs.editEntryFields[cs.editEntryIndex]
	switch ef.Type {
	case "string":
		ef.Value = raw
	case "int":
		v, err := strconv.Atoi(raw)
		if err != nil {
			return false
		}
		ef.Value = v
	default:
		return false
	}
	cs.applyEntryFieldAndMark(ef)
	return true
}
