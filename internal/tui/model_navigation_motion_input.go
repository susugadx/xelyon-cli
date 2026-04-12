package tui

import tea "github.com/charmbracelet/bubbletea"

func (m Model) handleNavigationCommandKey(msg tea.KeyMsg, s string) Model {
	if next, handled := m.handleNavigationVimKey(s); handled {
		return next
	}
	return m.handleNavigationScrollKey(msg)
}

func (m Model) handleNavigationScrollKey(msg tea.KeyMsg) Model {
	switch msg.Type {
	case tea.KeyUp:
		if m.focusedBlock >= 0 && len(m.toolBlocks) > 0 {
			m.moveBlockFocus(m.focusedBlock - 1)
		} else {
			m.moveCursorTo(m.cursorLine - 1)
		}
	case tea.KeyDown:
		if m.focusedBlock >= 0 && len(m.toolBlocks) > 0 {
			m.moveBlockFocus(m.focusedBlock + 1)
		} else {
			m.moveCursorTo(m.cursorLine + 1)
		}
	case tea.KeyPgUp:
		if m.focusedBlock < 0 {
			m.moveCursorTo(m.cursorLine - m.vp.height)
		}
	case tea.KeyPgDown:
		if m.focusedBlock < 0 {
			m.moveCursorTo(m.cursorLine + m.vp.height)
		}
	}

	return m
}
