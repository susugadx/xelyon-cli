package tui

import termtext "github.com/susugadx/xelyon-cli/internal/tui/termtext"

func (m Model) charSelectionColumnsForLine(line int) (startCol, endCol int, ok bool) {
	start, end, ok := m.normalizedCharSelection()
	if !ok || line < start.line || line > end.line {
		return 0, 0, false
	}

	plain := termtext.StripANSI(m.rawLines[line])
	lineWidth := termtext.PlainTextDisplayWidth(plain)
	switch {
	case start.line == end.line:
		return start.col, min(lineWidth, end.col+1), true
	case line == start.line:
		return start.col, lineWidth, true
	case line == end.line:
		return 0, min(lineWidth, end.col+1), true
	default:
		return 0, lineWidth, true
	}
}

func (m Model) viewportView() string {
	if m.hasActiveMouseSelection() {
		return m.renderViewportWithMouseSelection()
	}
	if !m.navigationMode || m.focusedBlock >= 0 {
		return m.vp.view()
	}
	return m.renderNavigationViewport()
}
