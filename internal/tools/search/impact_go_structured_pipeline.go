package search

import (
	"github.com/susugadx/xelyon-cli/internal/tools"
)

func tryStructuredGoImpactSearch(cache tools.ToolCacheInterface, opts SearchOptions) (string, bool) {
	result, ok := tryStructuredGoImpactSearchResult(cache, opts)
	if !ok {
		return "", false
	}
	return result.Rendered, true
}

func tryStructuredGoImpactSearchResult(cache tools.ToolCacheInterface, opts SearchOptions) (structuredImpactExecutionResult, bool) {
	ctx, scope, ok := newStructuredGoImpactSearchContext(opts)
	if !ok {
		return structuredImpactExecutionResult{}, false
	}
	return tryStructuredImpactSearchResult(cache, ctx, scope, resolveStructuredGoImpactSymbol)
}

func newStructuredGoImpactSearchContext(opts SearchOptions) (structuredImpactSearchContext, structuredImpactScope, bool) {
	return newStructuredImpactSearchContext(opts, structuredGoImpactRouteTag, normalizeStructuredGoImpactScope, structuredGoImpactRoute)
}

func structuredGoImpactRoute(pattern string, opts SearchOptions) (searchRouteTrace, bool) {
	return structuredImpactSymbolRoute(pattern, opts, "go", structuredGoImpactRouteTag)
}
