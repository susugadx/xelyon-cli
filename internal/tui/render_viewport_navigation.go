package tui

import "strings"

const (
	cursorLineBg   = "\033[48;5;236m"
	cursorCharBg   = "\033[48;5;255;38;5;16m"
	visualBg       = "\033[48;5;240m"
	visualCursorBg = "\033[48;5;255;38;5;16m"
)

func (m Model) renderNavigationViewport() string {
	visible := m.vp.visibleLines()
	var sb strings.Builder
	sb.Grow(m.vp.height * (m.vp.width + 1))

	for i := 0; i < m.vp.height; i++ {
		if i > 0 {
			sb.WriteByte('\n')
		}
		if i >= len(visible) {
			sb.WriteString(strings.Repeat(" ", max(0, m.vp.width)))
			continue
		}

		visIdx := m.vp.yOffset + i
		rawIdx := m.viewportRawLineIndex(visIdx)
		sb.WriteString(m.renderNavigationViewportLine(visible[i], visIdx, rawIdx))
	}

	return sb.String()
}

func (m Model) viewportRawLineIndex(visIdx int) int {
	rawIdx := visIdx
	if m.layout != nil && visIdx < len(m.layout.Rows) {
		rawIdx = m.layout.Rows[visIdx].RawLineIdx
	}
	return rawIdx
}

func (m Model) renderNavigationViewportLine(line string, visIdx int, rawIdx int) string {
	switch m.visualMode {
	case visualModeChar:
		if rendered, ok := m.renderVisualCharViewportLine(line, visIdx, rawIdx); ok {
			return rendered
		}
	case visualModeLine:
		if rendered, ok := m.renderVisualLineViewportLine(line, visIdx, rawIdx); ok {
			return rendered
		}
	}

	if rawIdx == m.cursorLine && isCursorInVisualRow(m.layout, visIdx, rawIdx, m.cursorCol) {
		return m.renderCursorViewportLine(line, visIdx, rawIdx, cursorCharBg, cursorLineBg)
	}
	return fitANSITextWidth(line, m.vp.width)
}
