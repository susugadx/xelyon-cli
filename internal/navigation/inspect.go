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

	_, output, status := resolveInspectSymbol(symbol, pathHint, inspectResolveOptions{
		budget: budget,
	})
	if status == SymbolAutoNone {
		return fmt.Sprintf("No symbol found: %q", symbol)
	}
	return output
}

func buildInspectResultForSingleCandidate(query symbolQuery, cand SymbolCandidate, opts inspectResolveOptions) InspectResult {
	result := InspectResult{Symbol: &cand}
	result.Body = readDefinitionBody(cand, opts.budget.BodyLines)

	allRefs, implementations, resolvedViaLSP, upstreamTruncated, upstreamIncomplete := collectInspectReferences(query.BaseName, cand, opts.referenceOptions())
	result.Implementations = implementations
	result.ResolvedViaLSP = resolvedViaLSP
	result.UpstreamTruncated = upstreamTruncated
	result.UpstreamIncomplete = upstreamIncomplete
	result.Callers, result.TotalCallers, result.MoreCallers = classifyCallers(allRefs, cand, opts.budget.CallerLimit)
	result.Refs, result.TotalRefs, result.MoreRefs = classifyRefs(allRefs, cand, opts.budget.RefLimit)
	result.Tests, result.TotalTests, result.MoreTests = findRelatedTests(query.BaseName, allRefs, opts.budget.TestLimit)

	if opts.normalizePaths {
		normalizeInspectResultPaths(&result, opts.runtime)
	}
	return result
}

func collectInspectReferences(baseName string, cand SymbolCandidate, opts inspectReferenceOptions) ([]Reference, []ImplementationRef, bool, bool, bool) {
	var implementations []ImplementationRef
	if opts.lspClient != nil {
		if cand.Kind == "interface" {
			if impls, implErr := findImplementationsViaLSP(opts.lspClient, cand, opts.runtime.InvocationCWD); implErr == nil {
				implementations = impls
			}
		}
		lspRefs, err := findReferencesViaLSP(opts.lspClient, cand, opts.runtime.InvocationCWD, opts.referenceFilter)
		if err == nil && len(lspRefs) > 0 {
			return lspRefs, implementations, true, false, false
		}
	}

	refs, truncated, incomplete := findReferencesWithFallbackRuntime(baseName, cand, opts.runtime, opts.referenceFilter, opts.fallbackSearchPath)
	return refs, implementations, false, truncated, incomplete
}
