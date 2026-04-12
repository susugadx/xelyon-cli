package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func (cs *configScreen) handleFieldSliceListKey(msg tea.KeyMsg, field *config.ConfigField) (configCommand, tea.Cmd) {
	s := msg.String()
	switch {
	case msg.Type == tea.KeyEsc:
		cs.closeFieldSliceEdit(field)
		return configCommandNone, nil
	case msg.Type == tea.KeyUp || s == "k":
		cs.moveSliceEditIndex(-1)
		return configCommandNone, nil
	case msg.Type == tea.KeyDown || s == "j":
		cs.moveSliceEditIndex(1)
		return configCommandNone, nil
	case s == "a":
		cs.beginSliceAdd()
		return configCommandNone, nil
	case s == "d":
		cs.deleteSelectedSliceItem()
		return configCommandNone, nil
	case isEnterKey(msg):
		cs.beginSliceItemEdit()
		return configCommandNone, nil
	default:
		return configCommandNone, nil
	}
}

func (cs *configScreen) handleEntrySliceListKey(msg tea.KeyMsg) (configCommand, tea.Cmd) {
	s := msg.String()
	switch {
	case msg.Type == tea.KeyEsc:
		cs.closeEntrySliceEdit()
		return configCommandNone, nil
	case msg.Type == tea.KeyUp || s == "k":
		cs.moveSliceEditIndex(-1)
		return configCommandNone, nil
	case msg.Type == tea.KeyDown || s == "j":
		cs.moveSliceEditIndex(1)
		return configCommandNone, nil
	case s == "a":
		cs.beginSliceAdd()
		return configCommandNone, nil
	case s == "d":
		cs.deleteSelectedSliceItem()
		return configCommandNone, nil
	case isEnterKey(msg):
		cs.beginSliceItemEdit()
		return configCommandNone, nil
	default:
		return configCommandNone, nil
	}
}

func (cs *configScreen) moveSliceEditIndex(delta int) {
	next := cs.editSliceIndex + delta
	if next < 0 {
		next = 0
	}
	if next >= len(cs.editSliceItems) {
		next = len(cs.editSliceItems) - 1
	}
	if next < 0 {
		next = 0
	}
	cs.editSliceIndex = next
}

func (cs *configScreen) closeFieldSliceEdit(field *config.ConfigField) {
	if err := config.SetFieldValue(cs.cfg, field.Path, cs.editSliceItems); err == nil && !sliceEqual(cs.editSliceItems, field.Current) {
		cs.markModified()
	}
	cs.editMode = editNone
}

func (cs *configScreen) closeEntrySliceEdit() {
	ef := &cs.editEntryFields[cs.editEntryIndex]
	items := make([]string, len(cs.editSliceItems))
	copy(items, cs.editSliceItems)
	ef.Value = items
	cs.applyEntryFieldAndMark(ef)
	cs.editEntryFieldEdit = ""
}
