package directquery

import (
	"github.com/susugadx/xelyon-cli/internal/tools"
)

// Plan centralizes gather_context-specific direct query policy
// so high-level routing does not need to know explicit/implicit path heuristics.
// Route ownership lives here: classify scoped direct, generic explicit direct,
// implicit direct, or search fallback. Exact scoped resolution itself lives in
// direct_query_resolve.go.
func Plan(execCtx tools.ExecutionContext, query string, policy Policy) Outcome {
	input, ok := parseDirectQueryInput(query)
	if !ok {
		return Outcome{Kind: OutcomeNone}
	}

	if inputHasOnlyExplicitPathSyntax(input) {
		return resolveExplicitRoute(execCtx, input, policy)
	}

	if hasScopedExactFilenameLookupScope(policy) {
		if outcome, handled := resolveScopedRoute(execCtx, input, policy); handled {
			return outcome
		}
	}

	return resolveFallbackRoute(execCtx, input, policy)
}
