package tui

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tui/termtext"
	"github.com/susugadx/xelyon-cli/internal/tui/theme"
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

	if rawIdx == m.cursorLine && termtext.CursorInVisualRow(m.layout, visIdx, rawIdx, m.cursorCol) {
		return m.renderCursorViewportLine(line, visIdx, rawIdx, theme.Viewport.CursorCharBg, theme.Viewport.CursorLineBg)
	}
	return fitANSITextWidth(line, m.vp.width)
}
