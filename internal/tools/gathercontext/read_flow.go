package gathercontext

import (
	"github.com/susugadx/xelyon-cli/internal/prompt"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/file/directquery"
	"github.com/susugadx/xelyon-cli/internal/tools/file/readtool"
)

func executeDirectRoute(execCtx tools.ExecutionContext, plan routePlan) executionResult {
	switch plan.kind {
	case routeLocatorRead:
		sections := readtool.ExecuteReadTargetsWithDetailSections(execCtx, plan.locatorQuery, "compact")
		return executionResult{
			routeHint:   "Direct read",
			direct:      &directExecution{body: readtool.RenderReadExecutionSections(sections)},
			observation: readtool.MergeReadExecutionSectionObservations(sections),
		}
	case routeDirect:
		body, observation := directquery.ExecuteWithObservation(execCtx, plan.direct.route, preferredDirectReadDetail(execCtx, plan.direct.route), 1)
		return executionResult{
			routeHint:   directRouteHint(plan.direct.route),
			direct:      &directExecution{body: body},
			observation: observation,
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

func directRouteHint(route directquery.Route) string {
	switch route.Kind {
	case directquery.RouteDirectory:
		return "Directory listing"
	default:
		return "Direct read"
	}
}

func directErrorHint() string {
	return "Direct query"
}

func preferredDirectReadDetail(execCtx tools.ExecutionContext, route directquery.Route) string {
	if prompt.ResolveEditToolModeWithConfig(execCtx.ProviderName, execCtx.Model, execCtx.Config) == prompt.EditToolModeApplyPatch && route.PrefersFullRead() {
		return "full"
	}
	return "auto"
}
