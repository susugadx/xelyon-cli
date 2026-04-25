package tui

import "github.com/susugadx/xelyon-cli/internal/tui/selection"

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
	start, end = selection.LineRange(m.visualStart.line, m.cursorLine)
	return start, end, true
}

func (m Model) normalizedCharSelection() (start, end visualPosition, ok bool) {
	if m.visualMode != visualModeChar || m.visualStart.line < 0 {
		return visualPosition{}, visualPosition{}, false
	}
	r, ok := selection.Normalize(m.visualStart.line, m.visualStart.col, m.cursorLine, m.cursorCol)
	if !ok {
		return visualPosition{}, visualPosition{}, false
	}
	return visualPosition{line: r.StartLine, col: r.StartCol}, visualPosition{line: r.EndLine, col: r.EndCol}, true
}
