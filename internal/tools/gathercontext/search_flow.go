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
		routeHint: searchRouteHint(plan, artifact),
		search: &searchExecution{
			discovery: artifact.Rendered,
		},
	}
	if prefetch := prefetchRecommendedEvidence(execCtx, artifact); prefetch != "" {
		result.routeHint = "Structured impact + prefetched evidence"
		result.search.prefetchedEvidence = prefetch
	}
	return result
}
