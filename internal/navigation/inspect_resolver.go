package navigation

import "github.com/susugadx/xelyon-cli/internal/locator"

type inspectResolveOptions struct {
	budget                      Budget
	runtime                     GoSymbolRuntime
	registry                    *locator.Registry
	lspClient                   LSPClient
	referenceFilter             ReferenceFilter
	fallbackReferenceSearchPath string
	normalizePaths              bool
}

type inspectReferenceOptions struct {
	runtime            GoSymbolRuntime
	lspClient          LSPClient
	referenceFilter    ReferenceFilter
	fallbackSearchPath string
}

func (opts inspectResolveOptions) referenceOptions() inspectReferenceOptions {
	return inspectReferenceOptions{
		runtime:            opts.runtime,
		lspClient:          opts.lspClient,
		referenceFilter:    opts.referenceFilter,
		fallbackSearchPath: opts.fallbackReferenceSearchPath,
	}
}

func resolveInspectSymbol(symbol, pathHint string, opts inspectResolveOptions) (InspectResult, string, SymbolAutoStatus) {
	query := parseSymbolQuery(symbol)
	candidates := resolveSymbolCandidatesWithRuntime(symbol, pathHint, opts.runtime)
	if len(candidates) == 0 {
		return InspectResult{}, "", SymbolAutoNone
	}
	if len(candidates) > 1 {
		result := InspectResult{Candidates: candidates}
		return result, formatMultipleCandidates(symbol, candidates, opts.registry), SymbolAutoMultiple
	}

	cand := candidates[0]
	result := buildInspectResultForSingleCandidate(query, cand, opts)
	return result, formatInspectResult(result, opts.registry), SymbolAutoSingle
}

func isZeroInspectBudget(budget Budget) bool {
	return budget.BodyLines == 0 && budget.CallerLimit == 0 && budget.RefLimit == 0 && budget.TestLimit == 0
}
