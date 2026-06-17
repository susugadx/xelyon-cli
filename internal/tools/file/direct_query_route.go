package file

import (
	"github.com/susugadx/xelyon-cli/internal/tools"
)

// PlanGatherContextDirectRoute centralizes gather_context-specific direct query policy
// so high-level routing does not need to know explicit/implicit path heuristics.
// Route ownership lives here: classify scoped direct, generic explicit direct,
// implicit direct, or search fallback. Exact scoped resolution itself lives in
// direct_query_resolve.go.
func PlanGatherContextDirectRoute(execCtx tools.ExecutionContext, query string, policy GatherContextDirectRoutePolicy) GatherContextDirectRouteOutcome {
	input, ok := parseDirectQueryInput(query)
	if !ok {
		return GatherContextDirectRouteOutcome{Kind: GatherContextDirectRouteOutcomeNone}
	}

	if inputHasOnlyExplicitPathSyntax(input) {
		return resolveExplicitGatherContextDirectRoute(execCtx, input, policy)
	}

	if hasScopedExactFilenameLookupScope(policy) {
		if outcome, handled := resolveScopedGatherContextDirectRoute(execCtx, input, policy); handled {
			return outcome
		}
	}

	return resolveFallbackGatherContextDirectRoute(execCtx, input, policy)
}
