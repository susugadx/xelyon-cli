package gathercontext

import (
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/search"
)

func executeSearchArtifact(execCtx tools.ExecutionContext, plan searchPlan) search.SearchExecutionArtifact {
	opts := buildSearchOptions(execCtx, plan)
	return search.ExecuteSearchCodeArtifactWithConfig(execCtx.EffectiveConfig(), execCtx.EffectiveToolCache(), opts)
}
