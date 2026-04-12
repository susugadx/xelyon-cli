package tui

func (m *Model) clearVisualSelection() {
	m.visualMode = visualModeOff
	m.visualStart = visualPosition{line: -1, col: -1}
	m.yPressed = false
}

func (m *Model) startCharVisualSelection() {
	m.clearMouseSelection()
	m.visualMode = visualModeChar
	m.visualStart = visualPosition{line: m.cursorLine, col: m.cursorCol}
	m.yPressed = false
	m.chromeDirty = true
}

func (m *Model) startLineVisualSelection() {
	m.clearMouseSelection()
	m.visualMode = visualModeLine
	m.visualStart = visualPosition{line: m.cursorLine, col: 0}
	m.yPressed = false
	m.chromeDirty = true
}

func (m Model) lineSelectionRange() (start, end int, ok bool) {
	if m.visualMode == visualModeOff || m.visualStart.line < 0 {
		return 0, 0, false
	}
	start = min(m.visualStart.line, m.cursorLine)
	end = max(m.visualStart.line, m.cursorLine)
	return start, end, true
}

func (m Model) normalizedCharSelection() (start, end visualPosition, ok bool) {
	if m.visualMode != visualModeChar || m.visualStart.line < 0 {
		return visualPosition{}, visualPosition{}, false
	}
	start = m.visualStart
	end = visualPosition{line: m.cursorLine, col: m.cursorCol}
	if start.line > end.line || (start.line == end.line && start.col > end.col) {
		start, end = end, start
	}
	return start, end, true
}
