package gathercontext

import "github.com/susugadx/xelyon-cli/internal/tools"

func executeRequestResult(execCtx tools.ExecutionContext, req request) executionResult {
	plan := buildRoutePlan(execCtx, req)
	return executePlan(execCtx, plan)
}

func executePlan(execCtx tools.ExecutionContext, plan routePlan) executionResult {
	switch plan.kind {
	case routeLocatorRead, routeDirect, routeDirectError:
		return executeDirectRoute(execCtx, plan)
	default:
		return executeSearchRoute(execCtx, plan.search)
	}
}
