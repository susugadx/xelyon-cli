package navigation

// SymbolAutoStatus はシンボル自動解決の結果種別。
type SymbolAutoStatus string

const (
	SymbolAutoSingle   SymbolAutoStatus = "single"
	SymbolAutoMultiple SymbolAutoStatus = "multiple"
	SymbolAutoNone     SymbolAutoStatus = "none"
)

// InspectSymbolAuto はシンボル名の自動解決を試みる。
// single: 単一候補が見つかった → summary 形式の結果を返す
// multiple: 複数候補 → 候補一覧を返す
// none: 見つからない → 空文字を返す（呼び出し側で text search にフォールバック）
func InspectSymbolAuto(symbol, pathHint string) (output string, status SymbolAutoStatus) {
	if symbol == "" {
		return "", SymbolAutoNone
	}

	query := parseSymbolQuery(symbol)
	candidates := resolveSymbolCandidates(symbol, pathHint)

	if len(candidates) == 0 {
		return "", SymbolAutoNone
	}

	if len(candidates) > 1 {
		return formatMultipleCandidates(symbol, candidates), SymbolAutoMultiple
	}

	// 単一候補 → SummaryBudget で結果を生成
	cand := candidates[0]
	result := InspectResult{Symbol: &cand}

	result.Body = readDefinitionBody(cand, SummaryBudget.BodyLines)

	ambiguousFiles := findAmbiguousFiles(query.BaseName, cand)
	allRefs, truncated, incomplete := findReferences(query.BaseName)
	result.UpstreamTruncated = truncated
	result.UpstreamIncomplete = incomplete
	allRefs = filterRefsByCandidate(allRefs, cand, ambiguousFiles)
	result.Callers, result.TotalCallers, result.MoreCallers = classifyCallers(allRefs, cand, SummaryBudget.CallerLimit)
	result.Refs, result.TotalRefs, result.MoreRefs = classifyRefs(allRefs, cand, SummaryBudget.RefLimit)
	result.Tests, result.TotalTests, result.MoreTests = findRelatedTests(query.BaseName, allRefs, SummaryBudget.TestLimit)

	return formatInspectResult(result), SymbolAutoSingle
}
