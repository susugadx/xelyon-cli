package tui

func (m Model) handleNavigationLineMotionKey(s string) (Model, bool) {
	switch s {
	case "j":
		if m.focusedBlock >= 0 && len(m.toolBlocks) > 0 {
			m.moveBlockFocus(m.focusedBlock + 1)
		} else {
			m.moveCursorTo(m.cursorLine + m.consumePendingCountOr(1))
		}
	case "k":
		if m.focusedBlock >= 0 && len(m.toolBlocks) > 0 {
			m.moveBlockFocus(m.focusedBlock - 1)
		} else {
			m.moveCursorTo(m.cursorLine - m.consumePendingCountOr(1))
		}
	case "h":
		if m.focusedBlock >= 0 {
			return m, true
		}
		m.moveCursorCol(-m.consumePendingCountOr(1))
	case "l":
		if m.focusedBlock >= 0 {
			return m, true
		}
		m.moveCursorCol(m.consumePendingCountOr(1))
	case "d":
		if m.focusedBlock >= 0 {
			return m, true
		}
		m.moveCursorTo(m.cursorLine + max(1, m.vp.height/2)*m.consumePendingCountOr(1))
	case "u":
		if m.focusedBlock >= 0 {
			return m, true
		}
		m.moveCursorTo(m.cursorLine - max(1, m.vp.height/2)*m.consumePendingCountOr(1))
	case "G":
		if m.focusedBlock >= 0 {
			return m, true
		}
		if m.pendingCount > 0 {
			m.moveCursorTo(m.pendingCount - 1)
			m.pendingCount = 0
		} else {
			m.moveCursorTo(len(m.rawLines) - 1)
		}
	case "0":
		if m.focusedBlock >= 0 {
			return m, true
		}
		m.moveCursorToLineStart(false, 1)
	case "^":
		if m.focusedBlock >= 0 {
			return m, true
		}
		m.moveCursorToLineStart(true, m.consumePendingCountOr(1))
	case "$":
		if m.focusedBlock >= 0 {
			return m, true
		}
		m.moveCursorToLineEnd(m.consumePendingCountOr(1))
	default:
		return m, false
	}

	return m, true
}
