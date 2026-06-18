package mutation

func shortestEditDistanceMyersWithLimit(oldLines, newLines []string, stepLimit int) (int, bool) {
	n := len(oldLines)
	m := len(newLines)
	maxDistance := n + m
	offset := maxDistance

	diagonalEnds := make([]int, 2*maxDistance+1)
	for i := range diagonalEnds {
		diagonalEnds[i] = -1
	}
	diagonalEnds[offset+1] = 0

	steps := 0
	for distance := 0; distance <= maxDistance; distance++ {
		for diagonal := -distance; diagonal <= distance; diagonal += 2 {
			steps++
			if stepLimit > 0 && steps > stepLimit {
				return 0, false
			}

			index := offset + diagonal
			var x int
			if diagonal == -distance || (diagonal != distance && diagonalEnds[index-1] < diagonalEnds[index+1]) {
				x = diagonalEnds[index+1]
			} else {
				x = diagonalEnds[index-1] + 1
			}

			y := x - diagonal
			for x < n && y < m && oldLines[x] == newLines[y] {
				x++
				y++
			}
			diagonalEnds[index] = x

			if x >= n && y >= m {
				return distance, true
			}
		}
	}
	return maxDistance, true
}
