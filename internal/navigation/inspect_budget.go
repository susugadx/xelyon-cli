package navigation

// ApplyInspectBudget は収集済みの inspect 結果に表示 budget を再適用する。
// 参照収集は重いため、impact routing では広めに集めた evidence から
// low/medium/high の表示上限だけを切り替える。
func ApplyInspectBudget(result InspectResult, budget Budget) InspectResult {
	if isZeroInspectBudget(budget) {
		budget = SummaryBudget
	}

	limited := result
	limited.Body = limitStrings(result.Body, budget.BodyLines)
	limited.Callers, limited.TotalCallers, limited.MoreCallers = limitReferences(result.Callers, result.TotalCallers, result.MoreCallers, budget.CallerLimit)
	limited.Refs, limited.TotalRefs, limited.MoreRefs = limitReferences(result.Refs, result.TotalRefs, result.MoreRefs, budget.RefLimit)
	limited.Tests, limited.TotalTests, limited.MoreTests = limitTestRefs(result.Tests, result.TotalTests, result.MoreTests, budget.TestLimit)
	return limited
}

func limitStrings(values []string, limit int) []string {
	if limit <= 0 || len(values) == 0 {
		return nil
	}
	if len(values) > limit {
		values = values[:limit]
	}
	return append([]string(nil), values...)
}

func limitReferences(refs []Reference, total int, more bool, limit int) ([]Reference, int, bool) {
	total = effectiveInspectTotal(total, len(refs))
	if limit <= 0 || len(refs) == 0 {
		return nil, total, total > 0 || more
	}
	if len(refs) > limit {
		refs = refs[:limit]
	}
	limited := append([]Reference(nil), refs...)
	return limited, total, total > len(limited) || more
}

func limitTestRefs(refs []TestRef, total int, more bool, limit int) ([]TestRef, int, bool) {
	total = effectiveInspectTotal(total, len(refs))
	if limit <= 0 || len(refs) == 0 {
		return nil, total, total > 0 || more
	}
	if len(refs) > limit {
		refs = refs[:limit]
	}
	limited := append([]TestRef(nil), refs...)
	return limited, total, total > len(limited) || more
}

func effectiveInspectTotal(total, collected int) int {
	if total < collected {
		return collected
	}
	return total
}
