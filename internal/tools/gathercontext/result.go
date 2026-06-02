package gathercontext

import (
	"github.com/susugadx/xelyon-cli/internal/tools"
	filetool "github.com/susugadx/xelyon-cli/internal/tools/file"
)

type routeKind string

const (
	routeLocatorRead routeKind = "locator_read"
	routeDirect      routeKind = "direct"
	routeDirectError routeKind = "direct_error"
	routeSearch      routeKind = "search"
)

type routePlan struct {
	kind         routeKind
	locatorQuery string
	direct       directPlan
	search       searchPlan
}

type executionResult struct {
	routeHint   string
	direct      *directExecution
	search      *searchExecution
	observation *tools.RuntimeObservation
}

type directPlan struct {
	route filetool.GatherContextDirectRoute
	err   string
}

type searchPlan struct {
	query          string
	path           string
	fileFilter     string
	preferImpact   bool
	literalPattern bool
}

type directExecution struct {
	body string
}

type searchExecution struct {
	discovery          string
	prefetchedEvidence string
}
