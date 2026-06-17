package search

import "github.com/susugadx/xelyon-cli/internal/tools"

type structuredImpactResolver func(symbol string, scope structuredImpactScope) symbolResolveResult
type structuredImpactOptionsNormalizer func(SearchOptions) (SearchOptions, bool)
type structuredImpactScopeNormalizer func(SearchOptions) (structuredImpactScope, bool)
type structuredImpactRoutePlanner func(pattern string, opts SearchOptions) (searchRouteTrace, bool)

type structuredImpactLanguageSpec struct {
	name               string
	routeTag           string
	normalize          structuredImpactScopeNormalizer
	planRoute          structuredImpactRoutePlanner
	resolver           structuredImpactResolver
	expandSupplemental bool
}

func structuredImpactLanguageSpecs() []structuredImpactLanguageSpec {
	return []structuredImpactLanguageSpec{
		structuredGoImpactLanguageSpec(),
		structuredTypeScriptImpactLanguageSpec(),
		structuredJavaScriptImpactLanguageSpec(),
	}
}

func (spec structuredImpactLanguageSpec) newSearchContext(opts SearchOptions) (structuredImpactSearchContext, structuredImpactScope, bool) {
	if spec.routeTag == "" || spec.normalize == nil || spec.planRoute == nil {
		return structuredImpactSearchContext{}, structuredImpactScope{}, false
	}
	return newStructuredImpactSearchContext(opts, spec.routeTag, spec.normalize, spec.planRoute)
}

func (spec structuredImpactLanguageSpec) trySearchResult(cache tools.ToolCacheInterface, opts SearchOptions) (structuredImpactExecutionResult, bool) {
	if spec.resolver == nil {
		return structuredImpactExecutionResult{}, false
	}
	ctx, scope, ok := spec.newSearchContext(opts)
	if !ok {
		return structuredImpactExecutionResult{}, false
	}
	return tryStructuredImpactSearchResult(cache, ctx, scope, spec.resolver)
}

func (spec structuredImpactLanguageSpec) trySearchResultForIntent(cache tools.ToolCacheInterface, opts SearchOptions) (structuredImpactExecutionResult, bool) {
	result, ok := spec.trySearchResult(cache, opts)
	if !ok {
		return structuredImpactExecutionResult{}, false
	}
	if spec.expandSupplemental {
		result = expandStructuredImpactSearchResult(cache, opts, result)
	}
	return result, true
}

func structuredImpactSameScope(opts SearchOptions) structuredImpactScope {
	return structuredImpactScope{
		Definition: opts,
		Evidence:   opts,
	}
}

func normalizeStructuredImpactSameScope(opts SearchOptions, normalize structuredImpactOptionsNormalizer) (structuredImpactScope, bool) {
	definition, ok := normalize(opts)
	if !ok {
		return structuredImpactScope{}, false
	}
	return structuredImpactSameScope(definition), true
}
