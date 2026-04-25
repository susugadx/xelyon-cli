package tui

import termtext "github.com/susugadx/xelyon-cli/internal/tui/termtext"

func (m *Model) clampCursorLine() {
	if len(m.rawLines) == 0 {
		m.cursorLine = 0
		m.cursorCol = 0
		return
	}
	if m.cursorLine < 0 {
		m.cursorLine = 0
	}
	if m.cursorLine >= len(m.rawLines) {
		m.cursorLine = len(m.rawLines) - 1
	}
	m.clampCursorCol()
}

func (m *Model) clampCursorCol() {
	maxCol := m.maxCursorColForLine(m.cursorLine)
	if m.cursorCol < 0 {
		m.cursorCol = 0
	}
	if m.cursorCol > maxCol {
		m.cursorCol = maxCol
	}
}

func (m Model) maxCursorColForLine(line int) int {
	if line < 0 || line >= len(m.rawLines) {
		return 0
	}
	width := termtext.PlainTextDisplayWidth(termtext.StripANSI(m.rawLines[line]))
	if width <= 0 {
		return 0
	}
	return width - 1
}

func (m *Model) consumePendingCountOr(fallback int) int {
	if m.pendingCount > 0 {
		count := m.pendingCount
		m.pendingCount = 0
		return count
	}
	return fallback
}
