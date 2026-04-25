package termtext

// CursorInVisualRow は raw cursor が指定 visual row に含まれるかを返す。
func CursorInVisualRow(layout *Layout, visIdx, rawLine, rawCol int) bool {
	if layout == nil {
		return true // Fallback without layout
	}
	rIdx, _ := layout.GetVisualCursor(rawLine, rawCol)
	return rIdx == visIdx
}

// LocalCursorCol は指定 visual row 内での cursor 表示列を返す。
func LocalCursorCol(layout *Layout, visIdx, rawLine, rawCol int) int {
	if layout == nil {
		return rawCol // Fallback without layout
	}
	rIdx, col := layout.GetVisualCursor(rawLine, rawCol)
	if rIdx != visIdx {
		return -1
	}
	return col
}
