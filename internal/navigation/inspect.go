package navigation

import "fmt"

// InspectSymbol は指定シンボルの定義・caller・ref・テストをまとめて返す。
func InspectSymbol(symbol, pathHint, mode string) string {
	if symbol == "" {
		return "Error: symbol is required"
	}

	inspectMode := ModeSummary
	if mode == "full" {
		inspectMode = ModeFull
	}

	budget := SummaryBudget
	if inspectMode == ModeFull {
		budget = FullBudget
	}

	_, output, status := resolveInspectSymbol(symbol, pathHint, budget, GoSymbolRuntime{}, nil, nil, nil, false)
	if status == SymbolAutoNone {
		return fmt.Sprintf("No symbol found: %q", symbol)
	}
	return output
}

func buildInspectResultForSingleCandidate(query symbolQuery, cand SymbolCandidate, budget Budget, runtime GoSymbolRuntime, lspClient LSPClient, referenceFilter ReferenceFilter, normalizePaths bool) InspectResult {
	result := InspectResult{Symbol: &cand}
	result.Body = readDefinitionBody(cand, budget.BodyLines)

	allRefs, implementations, resolvedViaLSP, upstreamTruncated, upstreamIncomplete := collectInspectReferences(query.BaseName, cand, runtime, lspClient, referenceFilter)
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

func collectInspectReferences(baseName string, cand SymbolCandidate, runtime GoSymbolRuntime, lspClient LSPClient, referenceFilter ReferenceFilter) ([]Reference, []ImplementationRef, bool, bool, bool) {
	var implementations []ImplementationRef
	if lspClient != nil {
		if cand.Kind == "interface" {
			if impls, implErr := findImplementationsViaLSP(lspClient, cand, runtime.InvocationCWD); implErr == nil {
				implementations = impls
			}
		}
		lspRefs, err := findReferencesViaLSP(lspClient, cand, runtime.InvocationCWD, referenceFilter)
		if err == nil && len(lspRefs) > 0 {
			return lspRefs, implementations, true, false, false
		}
	}

	refs, truncated, incomplete := findReferencesWithFallbackRuntime(baseName, cand, runtime, referenceFilter)
	return refs, implementations, false, truncated, incomplete
}
