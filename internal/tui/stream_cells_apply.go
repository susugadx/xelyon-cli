package tui

func streamCellStartIndex(cells []streamCell, idx int) int {
	if idx < 0 {
		return -1
	}
	if idx >= len(cells) {
		idx = len(cells) - 1
	}
	for idx >= 0 {
		if cells[idx].span > 0 {
			return idx
		}
		idx--
	}
	return -1
}

func applyClusterToStreamCells(cells []streamCell, col int, cluster string, width int, style string) []streamCell {
	if col < 0 {
		col = 0
	}
	if width <= 0 {
		start := streamCellStartIndex(cells, col-1)
		if start >= 0 {
			cells[start].text += cluster
			if style != "" && cells[start].style == "" {
				cells[start].style = style
			}
			return cells
		}
		if style != "" {
			return append(cells, streamCell{text: cluster, style: style, span: 1})
		}
		return append(cells, streamCell{text: cluster, span: 1})
	}

	for len(cells) < col {
		cells = append(cells, streamCell{text: " ", span: 1})
	}
	for len(cells) < col+width {
		cells = append(cells, streamCell{text: " ", span: 1})
	}

	for i := 0; i < width; i++ {
		target := col + i
		if target >= len(cells) {
			break
		}
		start := occupiedStreamCellStart(cells, target)
		span := occupiedStreamCellSpan(cells, start)
		for j := start; j < start+span && j < len(cells); j++ {
			cells[j] = streamCell{text: " ", style: cells[j].style, span: 1}
		}
	}

	cells[col] = streamCell{text: cluster, style: style, span: width}
	for i := col + 1; i < col+width; i++ {
		cells[i] = streamCell{style: style, span: 0}
	}
	return cells
}

func occupiedStreamCellStart(cells []streamCell, target int) int {
	if cells[target].span > 0 {
		return target
	}
	for start := target; start >= 0; start-- {
		if cells[start].span > 0 && start+cells[start].span > target {
			return start
		}
	}
	return target
}

func occupiedStreamCellSpan(cells []streamCell, start int) int {
	if start >= 0 && start < len(cells) && cells[start].span > 0 {
		return cells[start].span
	}
	return 1
}
