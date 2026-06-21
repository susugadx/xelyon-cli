package replaceengine

// BatchEditLineStats は batch replacement の追加/削除行数を表す。
type BatchEditLineStats struct {
	LinesAdded   int
	LinesRemoved int
}

// ResolveBatchExecutionLineStats は batch replacement の表示/record 用行数を解決する。
func ResolveBatchExecutionLineStats(oldContent, newContent string, edits []Edit, stdoutSuppressed bool) BatchEditLineStats {
	fallback := batchEditLineStats(edits)
	policy := resolveBatchDiffLineStatsPolicy(stdoutSuppressed)
	if exactLinesAdded, exactLinesRemoved, exact := resolveBatchDiffLineStatsWithPolicy(oldContent, newContent, policy); exact {
		return BatchEditLineStats{LinesAdded: exactLinesAdded, LinesRemoved: exactLinesRemoved}
	}

	return fallback
}
