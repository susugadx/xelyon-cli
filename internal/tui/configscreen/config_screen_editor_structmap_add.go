package configscreen

import (
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (cs *Screen) handleStructMapAddKey(msg tea.KeyMsg) (configCommand, tea.Cmd) {
	switch {
	case msg.Type == tea.KeyEsc:
		cs.editStructAdding = false
		cs.editStructInput.Blur()
		return configCommandNone, nil

	case isEnterKey(msg):
		key := strings.TrimSpace(cs.editStructInput.Value())
		if key != "" {
			cs.addStructMapKeyAndFocus(key)
		}
		cs.editStructAdding = false
		cs.editStructInput.Blur()
		return configCommandNone, nil

	default:
		var cmd tea.Cmd
		cs.editStructInput, cmd = cs.editStructInput.Update(msg)
		return configCommandNone, cmd
	}
}

func (cs *Screen) addStructMapKeyAndFocus(key string) {
	field := cs.selectedField()
	if field == nil || !cs.addStructMapKey(field.Path, key) {
		return
	}
	cs.editStructKeys = append(cs.editStructKeys, key)
	sort.Strings(cs.editStructKeys)
	cs.editStructIndex = sort.SearchStrings(cs.editStructKeys, key)
	cs.markModified()
}
