package termtext

// BuildLayout は raw lines を現在の表示幅に合わせた visual row 群へ変換する。
func BuildLayout(rawLines []string, width int) *Layout {
	if width <= 0 {
		width = 80
	}
	layout := &Layout{
		Width:        width,
		LineToRowMap: make([]int, len(rawLines)),
	}

	var rows []VisualRow
	var rowToLine []int

	for i, line := range rawLines {
		layout.LineToRowMap[i] = len(rows)
		if line == "" {
			rows = append(rows, VisualRow{RawLineIdx: i, SubRowIdx: 0, Content: "", Width: 0, PrefixWidth: 0})
			rowToLine = append(rowToLine, i)
			continue
		}

		wrapped := wrapLineToVisualRows(line, i, width)
		for _, row := range wrapped {
			rows = append(rows, row)
			rowToLine = append(rowToLine, i)
		}
	}

	layout.Rows = rows
	layout.RowToLineMap = rowToLine
	return layout
}
