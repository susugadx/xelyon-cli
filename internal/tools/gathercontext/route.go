package gathercontext

import (
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/file/directquery"
	"github.com/susugadx/xelyon-cli/internal/tools/search"
)

func buildRoutePlan(execCtx tools.ExecutionContext, req request) routePlan {
	req = normalizeRequest(req)
	plan := routePlan{
		kind:   routeSearch,
		search: newSearchPlan(req),
	}

	if !req.searchRouteIntent {
		if locatorPriority := classifyLocatorQuery(req.query, execCtx.EffectiveLocatorRegistry()); locatorPriority.ShouldRouteLocator() {
			plan.kind = routeLocatorRead
			plan.locatorQuery = req.query
			return plan
		}

		switch directOutcome := directquery.Plan(execCtx, req.query, directquery.Policy{
			AllowImplicitBareFile: allowImplicitBareFile(req),
			ScopedPath:            req.path,
			FileFilter:            req.fileFilter,
		}); directOutcome.Kind {
		case directquery.OutcomeResolved:
			plan.kind = routeDirect
			plan.direct.route = directOutcome.Route
			return plan
		case directquery.OutcomeError:
			plan.kind = routeDirectError
			plan.direct.err = directOutcome.Error
			return plan
		}
	}

	plan.search.preferImpact = !req.literalSearchPattern && search.ShouldPreferImpactIntent(plan.search.query)
	return plan
}

func newSearchPlan(req request) searchPlan {
	return searchPlan{
		query:          req.searchQuery,
		path:           req.searchPath,
		fileFilter:     req.fileFilter,
		literalPattern: req.literalSearchPattern,
	}
}

func allowImplicitBareFile(req request) bool {
	return req.path == "" && req.fileFilter == ""
}
