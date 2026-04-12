package tui

func (m Model) resolvePendingGotoLine(s string) (Model, bool) {
	if !m.gPressed {
		return m, false
	}

	m.gPressed = false
	if s != "g" || m.focusedBlock >= 0 {
		return m, false
	}

	targetLine := 0
	if m.pendingCount > 0 {
		targetLine = min(m.pendingCount-1, max(0, len(m.rawLines)-1))
		m.pendingCount = 0
	}
	m.moveCursorTo(targetLine)
	return m, true
}

func (m Model) handleNavigationCountPrefix(s string) (Model, bool) {
	if m.focusedBlock >= 0 || m.visualMode != visualModeOff || len(s) != 1 {
		return m, false
	}

	switch {
	case s[0] >= '1' && s[0] <= '9':
		m.pendingCount = m.pendingCount*10 + int(s[0]-'0')
		m.chromeDirty = true
		return m, true
	case s[0] == '0' && m.pendingCount > 0:
		m.pendingCount *= 10
		m.chromeDirty = true
		return m, true
	default:
		return m, false
	}
}
