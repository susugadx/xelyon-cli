package tui

import "github.com/susugadx/xelyon-cli/internal/tui/selection"

// hasActiveMouseSelection はマウス選択がアクティブかどうかを返す。
func (m Model) hasActiveMouseSelection() bool {
	return selection.Active(m.mouseSelAnchor.line, m.mouseSelAnchor.col, m.mouseSelEnd.line, m.mouseSelEnd.col)
}

// clearMouseSelection はマウス選択をクリアする。
func (m *Model) clearMouseSelection() {
	m.mouseSelAnchor = visualPosition{line: -1, col: -1}
	m.mouseSelEnd = visualPosition{line: -1, col: -1}
	m.mouseDragging = false
	m.mouseAutoScrolling = false
}

// normalizedMouseSelection は正規化されたマウス選択範囲を返す（start <= end）。
func (m Model) normalizedMouseSelection() (start, end visualPosition, ok bool) {
	if !m.hasActiveMouseSelection() {
		return visualPosition{}, visualPosition{}, false
	}
	r, ok := selection.Normalize(m.mouseSelAnchor.line, m.mouseSelAnchor.col, m.mouseSelEnd.line, m.mouseSelEnd.col)
	if !ok {
		return visualPosition{}, visualPosition{}, false
	}
	return visualPosition{line: r.StartLine, col: r.StartCol}, visualPosition{line: r.EndLine, col: r.EndCol}, true
}

// mouseSelectionColumnsForLine は指定行のマウス選択列範囲を返す。
// startCol は包含、endCol は排他。選択に含まれない行は ok=false を返す。
func (m Model) mouseSelectionColumnsForLine(line int) (startCol, endCol int, ok bool) {
	if !m.hasActiveMouseSelection() {
		return 0, 0, false
	}
	r, ok := selection.Normalize(m.mouseSelAnchor.line, m.mouseSelAnchor.col, m.mouseSelEnd.line, m.mouseSelEnd.col)
	if !ok {
		return 0, 0, false
	}
	return selection.ColumnsForLine(r, line, len(m.rawLines))
}

// mouseSelectionColumnsForVisualRow は rawLine レベルの選択列をビジュアル行ローカル列に変換する。
func (m Model) mouseSelectionColumnsForVisualRow(visIdx, rawIdx, startCol, endCol int) (int, int) {
	if m.layout == nil || visIdx >= len(m.layout.Rows) {
		return startCol, endCol
	}

	startVisRow, startVisCol := m.layout.GetVisualCursor(rawIdx, startCol)
	endVisRow, endVisCol := m.layout.GetVisualCursor(rawIdx, endCol)

	if endVisRow > startVisRow && endVisRow >= 0 && endVisRow < len(m.layout.Rows) {
		if endVisCol == m.layout.Rows[endVisRow].PrefixWidth {
			endVisRow--
			endVisCol = m.vp.width
		}
	}

	if visIdx < startVisRow || visIdx > endVisRow {
		return 0, 0
	}

	localStart := 0
	localEnd := m.vp.width
	if visIdx == startVisRow {
		localStart = startVisCol
	}
	if visIdx == endVisRow {
		localEnd = endVisCol
	}
	return localStart, localEnd
}

// screenToRawPosition はスクリーン座標を rawLine/rawCol に変換する。
// screenY は viewport 内の相対 Y 座標。
func (m Model) screenToRawPosition(screenX, screenY int) (rawLine, rawCol int, ok bool) {
	if m.layout == nil || len(m.layout.Rows) == 0 {
		return 0, 0, false
	}

	visRowIdx := m.vp.yOffset + screenY
	if visRowIdx < 0 {
		visRowIdx = 0
	}
	if visRowIdx >= len(m.layout.Rows) {
		visRowIdx = len(m.layout.Rows) - 1
	}

	row := m.layout.Rows[visRowIdx]
	rawLine = row.RawLineIdx

	baseCol := m.layout.GetRawColumnForVisualRow(visRowIdx)
	localX := screenX
	if localX < 0 {
		localX = 0
	}
	if localX < row.PrefixWidth {
		localX = row.PrefixWidth
	}
	rawCol = baseCol + (localX - row.PrefixWidth)

	maxCol := m.maxCursorColForLine(rawLine)
	if rawCol > maxCol {
		rawCol = maxCol
	}
	if rawCol < 0 {
		rawCol = 0
	}
	return rawLine, rawCol, true
}
