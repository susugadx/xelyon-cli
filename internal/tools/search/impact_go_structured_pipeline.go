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
	return structuredGoImpactLanguageSpec().trySearchResult(cache, opts)
}

func newStructuredGoImpactSearchContext(opts SearchOptions) (structuredImpactSearchContext, structuredImpactScope, bool) {
	return structuredGoImpactLanguageSpec().newSearchContext(opts)
}

func structuredGoImpactRoute(pattern string, opts SearchOptions) (searchRouteTrace, bool) {
	return structuredImpactSymbolRoute(pattern, opts, "go", structuredGoImpactRouteTag)
}

func structuredGoImpactLanguageSpec() structuredImpactLanguageSpec {
	return structuredImpactLanguageSpec{
		name:      "go",
		routeTag:  structuredGoImpactRouteTag,
		normalize: normalizeStructuredGoImpactScope,
		planRoute: structuredGoImpactRoute,
		resolver:  resolveStructuredGoImpactSymbol,
	}
}
