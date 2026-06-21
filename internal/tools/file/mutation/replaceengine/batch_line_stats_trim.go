package replaceengine

func trimSharedLineEdges(oldLines, newLines []string) ([]string, []string) {
	prefix := countCommonLinePrefix(oldLines, newLines)
	remainingOld := oldLines[prefix:]
	remainingNew := newLines[prefix:]

	suffix := countCommonLineSuffix(remainingOld, remainingNew)
	return remainingOld[:len(remainingOld)-suffix], remainingNew[:len(remainingNew)-suffix]
}

func countCommonLinePrefix(oldLines, newLines []string) int {
	limit := len(oldLines)
	if len(newLines) < limit {
		limit = len(newLines)
	}

	count := 0
	for count < limit && oldLines[count] == newLines[count] {
		count++
	}
	return count
}

func countCommonLineSuffix(oldLines, newLines []string) int {
	limit := len(oldLines)
	if len(newLines) < limit {
		limit = len(newLines)
	}

	count := 0
	for count < limit {
		oldIdx := len(oldLines) - 1 - count
		newIdx := len(newLines) - 1 - count
		if oldLines[oldIdx] != newLines[newIdx] {
			break
		}
		count++
	}
	return count
}
