package gathercontext

import (
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/search"
)

func executeSearchRoute(execCtx tools.ExecutionContext, plan searchPlan) executionResult {
	artifact := executeSearchArtifact(execCtx, plan)
	return buildSearchExecutionResult(execCtx, plan, artifact)
}

func searchRouteHint(plan searchPlan, artifact search.SearchExecutionArtifact) string {
	switch {
	case artifact.Metadata.StructuredImpact:
		return "Structured impact"
	case plan.preferImpact:
		return "Impact search"
	default:
		return "Auto search"
	}
}

func executeSearchArtifact(execCtx tools.ExecutionContext, plan searchPlan) search.SearchExecutionArtifact {
	opts := buildSearchOptions(execCtx, plan)
	return search.ExecuteSearchCodeArtifactWithConfig(execCtx.EffectiveConfig(), execCtx.EffectiveToolCache(), opts)
}

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
