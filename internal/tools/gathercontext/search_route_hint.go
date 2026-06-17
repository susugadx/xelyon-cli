package gathercontext

import "github.com/susugadx/xelyon-cli/internal/tools/search"

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
