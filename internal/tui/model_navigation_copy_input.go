package tui

func (m Model) resolvePendingNavigationCopy(s string) (Model, bool) {
	if !m.yPressed {
		return m, false
	}

	m.yPressed = false
	if m.hasActiveMouseSelection() {
		m.copyMouseSelection()
		return m, true
	}
	if s == "y" && m.focusedBlock < 0 && m.visualMode == visualModeOff {
		m.copyCursorLine()
		return m, true
	}
	if m.focusedBlock < 0 && m.visualMode == visualModeOff {
		m.copyDefaultSelectionTarget()
	}

	return m, false
}

func (m Model) handleNavigationCopyKey() Model {
	if m.hasActiveMouseSelection() {
		m.copyMouseSelection()
		return m
	}
	if m.visualMode != visualModeOff {
		m.copyVisualSelection()
		return m
	}
	if m.focusedBlock >= 0 && m.focusedBlock < len(m.toolBlocks) {
		m.copyFocusedBlock()
		return m
	}

	m.yPressed = true
	return m
}
