package configscreen

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (cs *Screen) handleSliceInputKey(msg tea.KeyMsg) (configCommand, tea.Cmd) {
	switch {
	case msg.Type == tea.KeyEsc:
		cs.finishSliceInput(false)
		return configCommandNone, nil
	case isEnterKey(msg):
		cs.finishSliceInput(true)
		return configCommandNone, nil
	default:
		var cmd tea.Cmd
		cs.editSliceInput, cmd = cs.editSliceInput.Update(msg)
		return configCommandNone, cmd
	}
}

func (cs *Screen) beginSliceAdd() {
	cs.editSliceAdding = true
	cs.editSliceInput.SetValue("")
	cs.editSliceInput.Focus()
}

func (cs *Screen) beginSliceItemEdit() {
	if cs.editSliceIndex < 0 || cs.editSliceIndex >= len(cs.editSliceItems) {
		return
	}
	cs.editSliceEditing = true
	cs.editSliceInput.SetValue(cs.editSliceItems[cs.editSliceIndex])
	cs.editSliceInput.Focus()
	cs.editSliceInput.CursorEnd()
}

func (cs *Screen) deleteSelectedSliceItem() {
	if cs.editSliceIndex < 0 || cs.editSliceIndex >= len(cs.editSliceItems) {
		return
	}
	cs.editSliceItems = append(cs.editSliceItems[:cs.editSliceIndex], cs.editSliceItems[cs.editSliceIndex+1:]...)
	if cs.editSliceIndex >= len(cs.editSliceItems) && cs.editSliceIndex > 0 {
		cs.editSliceIndex--
	}
}

func (cs *Screen) finishSliceInput(apply bool) {
	if apply {
		val := strings.TrimSpace(cs.editSliceInput.Value())
		if val != "" {
			if cs.editSliceAdding {
				cs.editSliceItems = append(cs.editSliceItems, val)
				cs.editSliceIndex = len(cs.editSliceItems) - 1
				cs.syncGuidanceChoicesForCurrentField()
			} else if cs.editSliceEditing && cs.editSliceIndex < len(cs.editSliceItems) {
				cs.editSliceItems[cs.editSliceIndex] = val
				cs.syncGuidanceChoicesForCurrentField()
			}
		}
	}
	cs.editSliceAdding = false
	cs.editSliceEditing = false
	cs.editSliceInput.Blur()
}
