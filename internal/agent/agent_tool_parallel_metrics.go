package agent

// recordSearchCodeBatchMerge は search_code batch merge をメトリクスに記録する。
// batch merge は tool_call + tool result が個別に履歴に残るため
// API 入力トークンの削減にはならない（実行最適化のみ）。
func (a *Agent) recordSearchCodeBatchMerge(mergedCount int) {
	a.statsMu.Lock()
	defer a.statsMu.Unlock()
	if a.Stats != nil {
		a.Stats.ToolObs.SearchCodeBatchMerges++
	}
}

// recordReadFileBatchMerge は read_file batch merge をメトリクスに記録する。
// batch merge は tool_call + tool result が個別に履歴に残るため
// API 入力トークンの削減にはならない（実行最適化のみ）。
func (a *Agent) recordReadFileBatchMerge(mergedCount int) {
	a.statsMu.Lock()
	defer a.statsMu.Unlock()
	if a.Stats != nil {
		a.Stats.ToolObs.ReadFileBatchMerges++
	}
}
