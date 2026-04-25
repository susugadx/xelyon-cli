package tui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/susugadx/xelyon-cli/internal/tui/termtext"
	"github.com/susugadx/xelyon-cli/internal/tui/theme"
)

func (m Model) renderVisualCharViewportLine(line string, visIdx int, rawIdx int) (string, bool) {
	startCol, endCol, ok := m.charSelectionColumnsForLine(rawIdx)
	if ok {
		plain := termtext.StripANSI(line)
		localStartCol, localEndCol := m.visualCharSelectionLocalRange(visIdx, rawIdx, startCol, endCol)
		if localStartCol < localEndCol {
			styled := termtext.StylePlainTextRange(plain, localStartCol, localEndCol, theme.Viewport.VisualBg)
			if rawIdx == m.cursorLine && termtext.CursorInVisualRow(m.layout, visIdx, rawIdx, m.cursorCol) {
				localCursorCol := termtext.LocalCursorCol(m.layout, visIdx, rawIdx, m.cursorCol)
				styled = termtext.StylePlainTextRangeWithCursor(plain, localStartCol, localEndCol, theme.Viewport.VisualBg, localCursorCol, theme.Viewport.VisualCursorBg, "")
			}
			return decorateViewportLine(styled, m.vp.width, ""), true
		}
	}

	if rawIdx == m.cursorLine && termtext.CursorInVisualRow(m.layout, visIdx, rawIdx, m.cursorCol) {
		return m.renderCursorViewportLine(line, visIdx, rawIdx, theme.Viewport.VisualCursorBg, ""), true
	}
	return "", false
}

func (m Model) visualCharSelectionLocalRange(visIdx int, rawIdx int, startCol int, endCol int) (localStartCol int, localEndCol int) {
	localStartCol = startCol
	localEndCol = endCol
	if m.layout == nil || rawIdx < 0 {
		return localStartCol, localEndCol
	}

	if m.isIntermediateCharSelectionLine(rawIdx) {
		return 0, 9999
	}

	startVisRow, startVisCol := m.layout.GetVisualCursor(rawIdx, startCol)
	endVisRow, endVisCol := m.layout.GetVisualCursor(rawIdx, endCol)

	if endVisRow > startVisRow && endVisRow >= 0 && endVisRow < len(m.layout.Rows) {
		if endVisCol == m.layout.Rows[endVisRow].PrefixWidth {
			endVisRow--
			endVisCol = 9999
		}
	}

	if visIdx < startVisRow || visIdx > endVisRow {
		return 0, 0
	}

	if visIdx == startVisRow {
		localStartCol = startVisCol
	} else {
		localStartCol = 0
	}
	if visIdx == endVisRow {
		localEndCol = endVisCol
	} else {
		localEndCol = 9999
	}
	return localStartCol, localEndCol
}

func (m Model) isIntermediateCharSelectionLine(rawIdx int) bool {
	return (rawIdx > m.visualStart.line && rawIdx < m.cursorLine) ||
		(rawIdx > m.cursorLine && rawIdx < m.visualStart.line)
}

func (m Model) renderVisualLineViewportLine(line string, visIdx int, rawIdx int) (string, bool) {
	start, end, ok := m.lineSelectionRange()
	if !ok || rawIdx < start || rawIdx > end {
		return "", false
	}

	if rawIdx == m.cursorLine && termtext.CursorInVisualRow(m.layout, visIdx, rawIdx, m.cursorCol) {
		plain := termtext.StripANSI(line)
		localCursorCol := termtext.LocalCursorCol(m.layout, visIdx, rawIdx, m.cursorCol)
		styled := termtext.StylePlainTextRangeWithCursor(plain, 0, lipgloss.Width(plain), theme.Viewport.VisualBg, localCursorCol, theme.Viewport.VisualCursorBg, "")
		return decorateViewportLine(styled, m.vp.width, theme.Viewport.VisualBg), true
	}
	return decorateViewportLine(line, m.vp.width, theme.Viewport.VisualBg), true
}

func (m Model) renderCursorViewportLine(line string, visIdx int, rawIdx int, cursorBg string, lineBg string) string {
	plain := termtext.StripANSI(line)
	localCursorCol := termtext.LocalCursorCol(m.layout, visIdx, rawIdx, m.cursorCol)
	styled := termtext.StylePlainTextRangeWithCursor(plain, 0, 0, "", localCursorCol, cursorBg, lineBg)
	return decorateViewportLine(styled, m.vp.width, lineBg)
}
