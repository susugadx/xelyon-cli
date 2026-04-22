package navigation

import "fmt"

// InspectSymbol は指定シンボルの定義・caller・ref・テストをまとめて返す。
func InspectSymbol(symbol, pathHint, mode string) string {
	if symbol == "" {
		return "Error: symbol is required"
	}

	query := parseSymbolQuery(symbol)

	inspectMode := ModeSummary
	if mode == "full" {
		inspectMode = ModeFull
	}

	budget := SummaryBudget
	if inspectMode == ModeFull {
		budget = FullBudget
	}

	// 1. シンボル候補を解決
	candidates := resolveSymbolCandidates(symbol, pathHint)
	if len(candidates) == 0 {
		return fmt.Sprintf("No symbol found: %q", symbol)
	}

	// 2. 複数候補 → 一覧のみ
	if len(candidates) > 1 {
		return formatMultipleCandidates(symbol, candidates, nil)
	}

	// 3. 単一候補 → 詳細取得
	cand := candidates[0]
	result := buildInspectResultForSingleCandidate(query, cand, budget, GoSymbolRuntime{}, nil, false)

	return formatInspectResult(result, nil)
}

func buildInspectResultForSingleCandidate(query symbolQuery, cand SymbolCandidate, budget Budget, runtime GoSymbolRuntime, lspClient LSPClient, normalizePaths bool) InspectResult {
	result := InspectResult{Symbol: &cand}
	result.Body = readDefinitionBody(cand, budget.BodyLines)

	allRefs, implementations, resolvedViaLSP, upstreamTruncated, upstreamIncomplete := collectInspectReferences(query.BaseName, cand, runtime, lspClient)
	result.Implementations = implementations
	result.ResolvedViaLSP = resolvedViaLSP
	result.UpstreamTruncated = upstreamTruncated
	result.UpstreamIncomplete = upstreamIncomplete
	result.Callers, result.TotalCallers, result.MoreCallers = classifyCallers(allRefs, cand, budget.CallerLimit)
	result.Refs, result.TotalRefs, result.MoreRefs = classifyRefs(allRefs, cand, budget.RefLimit)
	result.Tests, result.TotalTests, result.MoreTests = findRelatedTests(query.BaseName, allRefs, budget.TestLimit)

	if normalizePaths {
		normalizeInspectResultPaths(&result, runtime)
	}
	return result
}

func collectInspectReferences(baseName string, cand SymbolCandidate, runtime GoSymbolRuntime, lspClient LSPClient) ([]Reference, []ImplementationRef, bool, bool, bool) {
	var implementations []ImplementationRef
	if lspClient != nil {
		if cand.Kind == "interface" {
			if impls, implErr := findImplementationsViaLSP(lspClient, cand, runtime.InvocationCWD); implErr == nil {
				implementations = impls
			}
		}
		lspRefs, err := findReferencesViaLSP(lspClient, cand, runtime.InvocationCWD)
		if err == nil && len(lspRefs) > 0 {
			return lspRefs, implementations, true, false, false
		}
	}

	refs, truncated, incomplete := findReferencesWithFallbackRuntime(baseName, cand, runtime)
	return refs, implementations, false, truncated, incomplete
}
