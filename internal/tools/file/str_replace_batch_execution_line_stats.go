package file

func resolveBatchExecutionLineStats(oldContent, newContent string, edits []EditEntry, stdoutSuppressed bool) (linesAdded, linesRemoved int) {
	fallbackRemoved, fallbackAdded := batchEditLineStats(edits)
	policy := resolveBatchDiffLineStatsPolicy(stdoutSuppressed)
	if exactLinesAdded, exactLinesRemoved, exact := resolveBatchDiffLineStatsWithPolicy(oldContent, newContent, policy); exact {
		return exactLinesAdded, exactLinesRemoved
	}

	return fallbackAdded, fallbackRemoved
}
