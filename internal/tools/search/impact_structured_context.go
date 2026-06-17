package search

import "strings"

type structuredImpactSearchContext struct {
	Pattern  string
	Route    searchRouteTrace
	CacheKey string
}

type structuredImpactScope struct {
	Definition SearchOptions
	Evidence   SearchOptions
}

func newStructuredImpactSearchContext(opts SearchOptions, routeTag string, normalize structuredImpactScopeNormalizer, planRoute structuredImpactRoutePlanner) (structuredImpactSearchContext, structuredImpactScope, bool) {
	pattern := strings.TrimSpace(opts.Pattern)
	scope, ok := normalize(opts)
	if !shouldAttemptSinglePatternImpactSearch(opts, pattern) || !ok {
		return structuredImpactSearchContext{}, structuredImpactScope{}, false
	}

	route, ok := planRoute(pattern, opts)
	if !ok {
		return structuredImpactSearchContext{}, structuredImpactScope{}, false
	}

	return newStructuredImpactSearchContextForRoute(opts, pattern, route, routeTag), scope, true
}

func newStructuredImpactSearchContextForRoute(opts SearchOptions, pattern string, route searchRouteTrace, routeTag string) structuredImpactSearchContext {
	return structuredImpactSearchContext{
		Pattern:  pattern,
		Route:    route,
		CacheKey: buildStructuredImpactCacheKey(opts, route, routeTag),
	}
}

func buildStructuredImpactCacheKey(opts SearchOptions, route searchRouteTrace, routeTag string) string {
	return buildSearchCacheKeyWithRoute(opts, route.cacheSignature()+"|"+routeTag)
}

func shouldAttemptSinglePatternImpactSearch(opts SearchOptions, pattern string) bool {
	if !strings.EqualFold(strings.TrimSpace(opts.Intent), "impact") {
		return false
	}
	if len(splitPatterns(opts.Pattern)) != 1 {
		return false
	}
	return strings.TrimSpace(pattern) != ""
}
