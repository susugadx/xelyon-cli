package navigation

import "github.com/susugadx/xelyon-cli/internal/locator"

func resolveInspectSymbol(symbol, pathHint string, budget Budget, runtime GoSymbolRuntime, reg *locator.Registry, lspClient LSPClient, referenceFilter ReferenceFilter, normalizePaths bool) (InspectResult, string, SymbolAutoStatus) {
	query := parseSymbolQuery(symbol)
	candidates := resolveSymbolCandidatesWithRuntime(symbol, pathHint, runtime)
	if len(candidates) == 0 {
		return InspectResult{}, "", SymbolAutoNone
	}
	if len(candidates) > 1 {
		result := InspectResult{Candidates: candidates}
		return result, formatMultipleCandidates(symbol, candidates, reg), SymbolAutoMultiple
	}

	cand := candidates[0]
	result := buildInspectResultForSingleCandidate(query, cand, budget, runtime, lspClient, referenceFilter, normalizePaths)
	return result, formatInspectResult(result, reg), SymbolAutoSingle
}

func isZeroInspectBudget(budget Budget) bool {
	return budget.BodyLines == 0 && budget.CallerLimit == 0 && budget.RefLimit == 0 && budget.TestLimit == 0
}
