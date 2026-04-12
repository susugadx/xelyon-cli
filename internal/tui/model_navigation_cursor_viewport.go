package tui

func (m *Model) syncCursorToViewportTop() {
	if len(m.rawLines) == 0 {
		m.cursorLine = 0
		m.cursorCol = 0
		return
	}
	if m.layout != nil && m.vp.yOffset < len(m.layout.Rows) {
		m.cursorLine = m.layout.Rows[m.vp.yOffset].RawLineIdx
		m.cursorCol = m.layout.GetRawColumnForVisualRow(m.vp.yOffset)
	} else {
		m.cursorLine = 0
		m.cursorCol = 0
	}
	m.clampCursorCol()
}

func (m *Model) ensureCursorVisible() {
	m.clampCursorLine()
	if m.vp.height <= 0 {
		return
	}

	cursorRowIdx := -1
	if m.layout != nil {
		cursorRowIdx, _ = m.layout.GetVisualCursor(m.cursorLine, m.cursorCol)
	}
	if cursorRowIdx < 0 {
		return
	}

	if cursorRowIdx < m.vp.yOffset {
		m.vp.yOffset = cursorRowIdx
	}
	if cursorRowIdx >= m.vp.yOffset+m.vp.height {
		m.vp.yOffset = cursorRowIdx - m.vp.height + 1
	}
	if m.vp.yOffset > m.vp.maxYOffset() {
		m.vp.yOffset = m.vp.maxYOffset()
	}
	if m.vp.yOffset < 0 {
		m.vp.yOffset = 0
	}
	if m.vp.atBottom() {
		m.newOutput = false
	}
}
