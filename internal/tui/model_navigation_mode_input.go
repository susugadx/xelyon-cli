package tui

import tea "github.com/charmbracelet/bubbletea"

func (m Model) handleNavigationControlKey(msg tea.KeyMsg) (Model, bool) {
	if isEnterKey(msg) {
		return m.handleNavigationEnterKey(), true
	}

	switch msg.Type {
	case tea.KeyEsc:
		return m.handleNavigationEscapeKey(), true
	case tea.KeyTab:
		return m.handleNavigationTabKey(1), true
	case tea.KeyShiftTab:
		return m.handleNavigationTabKey(-1), true
	case tea.KeyUp:
		if m.focusedBlock >= 0 && len(m.toolBlocks) > 0 {
			m.moveBlockFocus(m.focusedBlock - 1)
			return m, true
		}
	case tea.KeyDown:
		if m.focusedBlock >= 0 && len(m.toolBlocks) > 0 {
			m.moveBlockFocus(m.focusedBlock + 1)
			return m, true
		}
	}

	return m, false
}

func (m Model) handleNavigationEnterKey() Model {
	if m.focusedBlock >= 0 && m.focusedBlock < len(m.toolBlocks) {
		m.toggleToolBlock(m.focusedBlock)
		return m
	}

	m.switchToComposerInput()
	return m
}

func (m Model) handleNavigationEscapeKey() Model {
	if m.hasActiveMouseSelection() {
		m.clearMouseSelection()
		m.chromeDirty = true
		return m
	}
	if m.visualMode != visualModeOff {
		m.clearVisualSelection()
		m.chromeDirty = true
		return m
	}
	if m.focusedBlock >= 0 {
		m.clearBlockFocus()
		m.chromeDirty = true
		return m
	}

	m.switchToComposerInput()
	return m
}

func (m Model) handleNavigationTabKey(delta int) Model {
	if m.visualMode != visualModeOff || len(m.toolBlocks) == 0 {
		return m
	}

	if m.focusedBlock >= 0 && m.focusedBlock < len(m.toolBlocks) {
		m.moveBlockFocus(m.focusedBlock + delta)
		return m
	}
	if delta > 0 {
		m.setBlockFocus(len(m.toolBlocks) - 1)
		return m
	}

	m.setBlockFocus(0)
	return m
}

func (m Model) handleNavigationModeRune(s string) (Model, bool) {
	switch s {
	case "q", "i":
		m.switchToComposerInput()
		return m, true
	default:
		return m, false
	}
}
