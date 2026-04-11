package gathercontext

import (
	"github.com/susugadx/xelyon-cli/internal/tools"
	filetool "github.com/susugadx/xelyon-cli/internal/tools/file"
	"github.com/susugadx/xelyon-cli/internal/tools/search"
)

func buildRoutePlan(execCtx tools.ExecutionContext, req request) routePlan {
	req = normalizeRequest(req)
	plan := routePlan{
		kind:   routeSearch,
		search: newSearchPlan(req),
	}

	if locatorPriority := classifyLocatorQuery(req.query, execCtx.EffectiveLocatorRegistry()); locatorPriority.ShouldRouteLocator() {
		plan.kind = routeLocatorRead
		plan.locatorQuery = req.query
		return plan
	}

	switch directOutcome := filetool.PlanGatherContextDirectRoute(execCtx, req.query, filetool.GatherContextDirectRoutePolicy{
		AllowImplicitBareFile: allowImplicitBareFile(req),
		ScopedPath:            req.path,
		FileFilter:            req.fileFilter,
	}); directOutcome.Kind {
	case filetool.GatherContextDirectRouteOutcomeResolved:
		plan.kind = routeDirect
		plan.direct.route = directOutcome.Route
		return plan
	case filetool.GatherContextDirectRouteOutcomeError:
		plan.kind = routeDirectError
		plan.direct.err = directOutcome.Error
		return plan
	}

	plan.search.preferImpact = search.ShouldPreferImpactIntent(plan.search.query)
	return plan
}

func newSearchPlan(req request) searchPlan {
	return searchPlan{
		query:      req.query,
		path:       req.path,
		fileFilter: req.fileFilter,
	}
}

func allowImplicitBareFile(req request) bool {
	return req.path == "" && req.fileFilter == ""
}
