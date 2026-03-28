package tui

func isCursorInVisualRow(layout *Layout, visIdx, rawLine, rawCol int) bool {
	if layout == nil {
		return true // Fallback without layout
	}
	rIdx, _ := layout.GetVisualCursor(rawLine, rawCol)
	return rIdx == visIdx
}

func getLocalCursorCol(layout *Layout, visIdx, rawLine, rawCol int) int {
	if layout == nil {
		return rawCol // Fallback without layout
	}
	rIdx, col := layout.GetVisualCursor(rawLine, rawCol)
	if rIdx != visIdx {
		return -1
	}
	return col
}
