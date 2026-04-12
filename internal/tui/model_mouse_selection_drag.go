package tui

import tea "github.com/charmbracelet/bubbletea"

func (m *Model) handleMouseSelectionPress(msg tea.MouseMsg) tea.Cmd {
	if msg.Button != tea.MouseButtonLeft || msg.Y >= m.vp.height {
		return nil
	}
	rawLine, rawCol, ok := m.screenToRawPosition(msg.X, msg.Y)
	if !ok {
		return nil
	}
	if m.visualMode != visualModeOff {
		m.clearVisualSelection()
	}
	m.resetNavPending()
	m.mouseSelAnchor = visualPosition{line: rawLine, col: rawCol}
	m.mouseSelEnd = m.mouseSelAnchor
	m.mouseDragging = true
	m.mouseAutoScrolling = false
	m.mouseLastScreenX = msg.X
	m.mouseLastScreenY = msg.Y
	m.chromeDirty = true
	return nil
}

func (m *Model) handleMouseSelectionMotion(msg tea.MouseMsg) tea.Cmd {
	if !m.mouseDragging {
		return nil
	}
	m.mouseLastScreenX = msg.X
	m.mouseLastScreenY = msg.Y
	m.updateMouseSelectionEnd(msg.X, msg.Y)
	m.chromeDirty = true

	atEdge := msg.Y < autoScrollEdgeZone || msg.Y >= m.vp.height-autoScrollEdgeZone
	if atEdge && !m.mouseAutoScrolling {
		m.mouseAutoScrolling = true
		return mouseAutoScrollTick()
	}
	if !atEdge {
		m.mouseAutoScrolling = false
	}
	return nil
}

func (m *Model) handleMouseSelectionRelease(msg tea.MouseMsg) tea.Cmd {
	if !m.mouseDragging {
		return nil
	}
	m.mouseDragging = false
	m.mouseAutoScrolling = false
	m.updateMouseSelectionEnd(msg.X, msg.Y)
	if m.mouseSelAnchor == m.mouseSelEnd {
		m.clearMouseSelection()
	}
	m.chromeDirty = true
	return nil
}

func (m *Model) updateMouseSelectionEnd(screenX, screenY int) {
	clampedY := screenY
	if clampedY < 0 {
		clampedY = 0
	}
	if clampedY >= m.vp.height {
		clampedY = m.vp.height - 1
	}
	if rawLine, rawCol, ok := m.screenToRawPosition(screenX, clampedY); ok {
		m.mouseSelEnd = visualPosition{line: rawLine, col: rawCol}
	}
}
