package tui

func (m Model) handleNavigationWordMotionKey(s string) (Model, bool) {
	switch s {
	case "w":
		if m.focusedBlock >= 0 {
			return m, true
		}
		m.moveCursorToNextWordStart(m.consumePendingCountOr(1))
	case "b":
		if m.focusedBlock >= 0 {
			return m, true
		}
		m.moveCursorToPrevWordStart(m.consumePendingCountOr(1))
	case "e":
		if m.focusedBlock >= 0 {
			return m, true
		}
		m.moveCursorToWordEnd(m.consumePendingCountOr(1))
	default:
		return m, false
	}

	return m, true
}
