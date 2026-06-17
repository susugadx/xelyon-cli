package gathercontext

import (
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/search"
)

func buildSearchExecutionResult(execCtx tools.ExecutionContext, plan searchPlan, artifact search.SearchExecutionArtifact) executionResult {
	result := executionResult{
		routeHint:   searchRouteHint(plan, artifact),
		observation: tools.CloneRuntimeObservation(artifact.Metadata.Observation),
		search: &searchExecution{
			discovery: artifact.Rendered,
		},
	}
	prefetch := prefetchRecommendedEvidence(execCtx, artifact)
	if prefetch.discoveryNote != "" {
		result.search.discovery = appendSearchDiscoveryNote(result.search.discovery, prefetch.discoveryNote)
	}
	if prefetch.output != "" {
		result.routeHint = "Structured impact + prefetched evidence"
		result.search.prefetchedEvidence = prefetch.output
		result.observation = tools.MergeRuntimeObservations(result.observation, prefetch.observation)
	}
	return result
}
