package configscreen

import (
	"strconv"

	tea "github.com/charmbracelet/bubbletea"
)

func (cs *Screen) handleStructEntryEdit(msg tea.KeyMsg) (configCommand, tea.Cmd) {
	if cs.editEntryFieldEdit != "" {
		return cs.handleStructEntryFieldEdit(msg)
	}

	s := msg.String()
	switch {
	case msg.Type == tea.KeyEsc:
		cs.closeStructEntryEdit()
		return configCommandNone, nil

	case msg.Type == tea.KeyUp || s == "k":
		if cs.editEntryIndex > 0 {
			cs.editEntryIndex--
		}
		return configCommandNone, nil

	case msg.Type == tea.KeyDown || s == "j":
		if cs.editEntryIndex < len(cs.editEntryFields)-1 {
			cs.editEntryIndex++
		}
		return configCommandNone, nil

	case s == " ":
		cs.toggleSelectedStructEntryBool()
		return configCommandNone, nil

	case isEnterKey(msg):
		cs.activateSelectedStructEntryField()
		return configCommandNone, nil
	}
	return configCommandNone, nil
}

func (cs *Screen) closeStructEntryEdit() {
	cs.editEntryActive = false
	cs.editEntryFields = nil
}

func (cs *Screen) toggleSelectedStructEntryBool() {
	if cs.editEntryIndex >= len(cs.editEntryFields) {
		return
	}
	ef := &cs.editEntryFields[cs.editEntryIndex]
	if ef.Type != "bool" {
		return
	}
	cur, _ := ef.Value.(bool)
	updated := *ef
	updated.Value = !cur
	if cs.applyEntryFieldAndMark(updated) {
		*ef = updated
	}
}

func (cs *Screen) activateSelectedStructEntryField() {
	if cs.editEntryIndex < 0 || cs.editEntryIndex >= len(cs.editEntryFields) {
		return
	}
	ef := &cs.editEntryFields[cs.editEntryIndex]
	switch ef.Type {
	case "bool":
		cur, _ := ef.Value.(bool)
		updated := *ef
		updated.Value = !cur
		if cs.applyEntryFieldAndMark(updated) {
			*ef = updated
		}
	case "string":
		cs.beginStructEntryInputEdit(ef, func(value interface{}) string {
			s, _ := value.(string)
			return s
		})
	case "int":
		cs.beginStructEntryInputEdit(ef, func(value interface{}) string {
			n, _ := value.(int)
			return strconv.Itoa(n)
		})
	case "[]string":
		cs.beginStructEntrySliceEdit(ef)
	}
}

func (cs *Screen) beginStructEntryInputEdit(ef *structEntryField, format func(interface{}) string) {
	cs.editEntryFieldEdit = "input"
	cs.editInput.SetValue(format(ef.Value))
	cs.editInput.Focus()
	cs.editInput.CursorEnd()
}

func (cs *Screen) beginStructEntrySliceEdit(ef *structEntryField) {
	cs.editEntryFieldEdit = "slice"
	if s, ok := ef.Value.([]string); ok {
		cs.editSliceItems = make([]string, len(s))
		copy(cs.editSliceItems, s)
	} else {
		cs.editSliceItems = nil
	}
	cs.editSliceIndex = 0
	cs.editSliceAdding = false
	cs.editSliceEditing = false
}
