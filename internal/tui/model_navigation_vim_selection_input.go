package tui

func (m Model) handleNavigationSelectionKey(s string) (Model, bool) {
	switch s {
	case "g":
		if m.focusedBlock < 0 {
			m.gPressed = true
		}
	case "v":
		if m.focusedBlock < 0 {
			m.startCharVisualSelection()
		}
	case "V":
		if m.focusedBlock < 0 {
			m.startLineVisualSelection()
		}
	case "y":
		return m.handleNavigationCopyKey(), true
	default:
		return m, false
	}

	return m, true
}
