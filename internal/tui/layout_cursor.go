package tui

// GetRawColumnForVisualRow は指定 visual row の先頭に対応する raw column を返す。
func (l *Layout) GetRawColumnForVisualRow(visRowIdx int) int {
	if visRowIdx < 0 || visRowIdx >= len(l.Rows) {
		return 0
	}
	rawLine := l.Rows[visRowIdx].RawLineIdx
	startRow := l.LineToRowMap[rawLine]
	rawCol := 0
	for i := startRow; i < visRowIdx; i++ {
		rawCol += l.Rows[i].Width - l.Rows[i].PrefixWidth
	}
	return rawCol
}

// GetVisualCursor は raw line / raw column を表示上の row / column へ変換する。
func (l *Layout) GetVisualCursor(rawLine, rawCol int) (rowIdx, col int) {
	if rawLine < 0 || rawLine >= len(l.LineToRowMap) {
		return -1, -1
	}
	startRow := l.LineToRowMap[rawLine]
	endRow := len(l.Rows)
	if rawLine+1 < len(l.LineToRowMap) {
		endRow = l.LineToRowMap[rawLine+1]
	}
	if startRow >= endRow {
		return -1, -1
	}

	remainingCol := rawCol
	for i := startRow; i < endRow; i++ {
		rowWidth := l.Rows[i].Width
		prefixWidth := l.Rows[i].PrefixWidth
		contentWidth := rowWidth - prefixWidth
		if remainingCol <= contentWidth {
			if remainingCol < contentWidth || i == endRow-1 {
				return i, prefixWidth + remainingCol
			}
		}
		remainingCol -= contentWidth
	}
	return endRow - 1, l.Rows[endRow-1].Width
}
