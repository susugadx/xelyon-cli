package tui

import tea "github.com/charmbracelet/bubbletea"

func (cs *configScreen) handleStructEntryFieldEdit(msg tea.KeyMsg) (configCommand, tea.Cmd) {
	switch cs.editEntryFieldEdit {
	case "input":
		return cs.handleEntryInputEdit(msg)
	case "slice":
		return cs.handleEntrySliceEdit(msg)
	}
	return configCommandNone, nil
}

// sliceEqual は []string の等値比較。
func sliceEqual(a []string, b interface{}) bool {
	bs, ok := b.([]string)
	if !ok {
		return false
	}
	if len(a) != len(bs) {
		return false
	}
	for i := range a {
		if a[i] != bs[i] {
			return false
		}
	}
	return true
}
