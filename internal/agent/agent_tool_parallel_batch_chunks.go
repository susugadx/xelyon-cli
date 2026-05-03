package agent

func forEachIndexChunk(total, chunkSize int, fn func(start, end int)) {
	if total <= 0 || chunkSize <= 0 || fn == nil {
		return
	}
	for start := 0; start < total; start += chunkSize {
		end := start + chunkSize
		if end > total {
			end = total
		}
		fn(start, end)
	}
}

func nextPathBudgetChunk(pathCounts []int, start, maxPaths int) (end int, totalPaths int) {
	if start >= len(pathCounts) || maxPaths <= 0 {
		return start, 0
	}
	end = start
	for end < len(pathCounts) {
		nextCount := pathCounts[end]
		if totalPaths > 0 && totalPaths+nextCount > maxPaths {
			break
		}
		totalPaths += nextCount
		end++
	}
	return end, totalPaths
}
