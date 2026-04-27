package gathercontext

import (
	"github.com/susugadx/xelyon-cli/internal/prompt"
	"github.com/susugadx/xelyon-cli/internal/tools"
	filetool "github.com/susugadx/xelyon-cli/internal/tools/file"
)

func executeDirectRoute(execCtx tools.ExecutionContext, plan routePlan) executionResult {
	switch plan.kind {
	case routeLocatorRead:
		return executionResult{
			routeHint: "Direct read",
			direct: &directExecution{
				body: filetool.ExecuteReadTargetsWithDetail(execCtx, plan.locatorQuery, "compact"),
			},
		}
	case routeDirect:
		return executionResult{
			routeHint: directRouteHint(plan.direct.route),
			direct: &directExecution{
				body: filetool.ExecuteGatherContextDirectRoute(execCtx, plan.direct.route, preferredDirectReadDetail(execCtx, plan.direct.route), 1),
			},
		}
	case routeDirectError:
		return executionResult{
			routeHint: directErrorHint(),
			direct: &directExecution{
				body: plan.direct.err,
			},
		}
	default:
		return executionResult{}
	}
}

func directRouteHint(route filetool.GatherContextDirectRoute) string {
	switch route.Kind {
	case filetool.GatherContextDirectRouteDirectory:
		return "Directory listing"
	default:
		return "Direct read"
	}
}

func directErrorHint() string {
	return "Direct query"
}

func preferredDirectReadDetail(execCtx tools.ExecutionContext, route filetool.GatherContextDirectRoute) string {
	if prompt.ResolveEditToolModeWithConfig(execCtx.ProviderName, execCtx.Model, execCtx.Config) == prompt.EditToolModeApplyPatch && route.PrefersFullRead() {
		return "full"
	}
	return "auto"
}
