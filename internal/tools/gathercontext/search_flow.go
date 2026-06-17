package gathercontext

import (
	"github.com/susugadx/xelyon-cli/internal/tools"
)

func executeSearchRoute(execCtx tools.ExecutionContext, plan searchPlan) executionResult {
	artifact := executeSearchArtifact(execCtx, plan)
	return buildSearchExecutionResult(execCtx, plan, artifact)
}
