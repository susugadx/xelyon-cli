package search

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

type structuredImpactSearchContext struct {
	Pattern  string
	Route    searchRouteTrace
	CacheKey string
}

type structuredImpactScope struct {
	Definition SearchOptions
	Evidence   SearchOptions
}

type structuredImpactCachedResult struct {
	Output        string
	Bundle        *SymbolBundle
	AffectedFiles []string
	Observation   *tools.RuntimeObservation
}

type structuredImpactExecutionResult struct {
	Rendered      string
	Bundle        *SymbolBundle
	AffectedFiles []string
	Observation   *tools.RuntimeObservation
	Ambiguous     bool
	MultiPattern  bool
}

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

func tryStructuredImpactSearchResultForIntent(cache tools.ToolCacheInterface, opts SearchOptions) (structuredImpactExecutionResult, bool) {
	for _, spec := range structuredImpactLanguageSpecs() {
		if result, ok := spec.trySearchResultForIntent(cache, opts); ok {
			return result, true
		}
	}
	return structuredImpactExecutionResult{}, false
}

func tryStructuredImpactSearchResult(cache tools.ToolCacheInterface, ctx structuredImpactSearchContext, scope structuredImpactScope, resolver structuredImpactResolver) (structuredImpactExecutionResult, bool) {
	runtimeOpts := scope.Definition
	if cached, ok := loadStructuredImpactCachedResult(cache, ctx, runtimeOpts); ok {
		return structuredImpactExecutionResult{
			Rendered:      cached.Output,
			Bundle:        cached.Bundle,
			AffectedFiles: cached.AffectedFiles,
			Observation:   cached.Observation,
			Ambiguous:     cached.Bundle == nil,
		}, true
	}

	return resolveStructuredImpactWithContext(cache, ctx, scope, resolver)
}

func loadStructuredImpactCachedResult(cache tools.ToolCacheInterface, ctx structuredImpactSearchContext, opts SearchOptions) (structuredImpactCachedResult, bool) {
	if cache == nil {
		return structuredImpactCachedResult{}, false
	}
	cached, ok := cache.GetSearch(ctx.Pattern, ctx.CacheKey)
	if !ok {
		return structuredImpactCachedResult{}, false
	}

	bundle := loadSinglePatternBundle(ctx.Pattern, ctx.CacheKey)
	bundle, cached = formatImpactBundleForRuntimeWithContext(bundle, cached, opts, cache, currentSearchImpactRuntimeRankContext(ctx.Pattern, ctx.CacheKey))
	if bundle == nil && !isStructuredImpactAmbiguousOutput(cached) {
		return structuredImpactCachedResult{}, false
	}
	if bundle != nil && !isValidStructuredImpactSingleBundle(bundle) {
		return structuredImpactCachedResult{}, false
	}
	affectedFiles := loadSearchAffectedFiles(ctx.Pattern, ctx.CacheKey)
	if len(affectedFiles) == 0 {
		affectedFiles = deriveAffectedFilesFromCachedResult(bundle, cached, opts)
	}

	return structuredImpactCachedResult{
		Output:        cached,
		Bundle:        bundle,
		AffectedFiles: affectedFiles,
		Observation:   loadCachedStructuredImpactObservation(ctx, bundle, opts),
	}, true
}

func loadCachedStructuredImpactObservation(ctx structuredImpactSearchContext, bundle *SymbolBundle, opts SearchOptions) *tools.RuntimeObservation {
	if bundle != nil {
		return observationForSymbolBundle(bundle, opts)
	}
	if observation := loadSinglePatternObservation(ctx.Pattern, ctx.CacheKey); observation != nil {
		return observation
	}
	return nil
}

func resolveStructuredImpactWithContext(cache tools.ToolCacheInterface, ctx structuredImpactSearchContext, scope structuredImpactScope, resolver structuredImpactResolver) (structuredImpactExecutionResult, bool) {
	if resolver == nil {
		return structuredImpactExecutionResult{}, false
	}

	runtimeOpts := scope.Definition
	resolved := resolver(structuredImpactResolverSymbol(ctx), scope)
	route := ctx.Route
	route.SymbolAttempted = true

	switch resolved.Status {
	case symbolResolveSingle:
		return resolveStructuredImpactSingleResult(cache, ctx, runtimeOpts, route, resolved)
	case symbolResolveMultiple:
		return resolveStructuredImpactMultipleResult(cache, ctx, runtimeOpts, resolved), true
	default:
		return structuredImpactExecutionResult{}, false
	}
}

func structuredImpactResolverSymbol(ctx structuredImpactSearchContext) string {
	if symbol := strings.TrimSpace(ctx.Route.SymbolQuery); symbol != "" {
		return symbol
	}
	return ctx.Pattern
}

func resolveStructuredImpactSingleResult(cache tools.ToolCacheInterface, ctx structuredImpactSearchContext, opts SearchOptions, route searchRouteTrace, resolved symbolResolveResult) (structuredImpactExecutionResult, bool) {
	if !isValidStructuredImpactSingleBundle(resolved.Bundle) {
		return structuredImpactExecutionResult{}, false
	}

	route.SymbolResolved = true
	route.FinalLane = searchLaneSymbol
	resolved.Bundle = attachBundleRoute(resolved.Bundle, route)
	affectedFiles := collectSymbolBundleAffectedFiles(resolved.Bundle, opts)
	outputBundle, output := formatImpactBundleForRuntime(resolved.Bundle, resolved.Output, opts, cache)
	observation := observationForSymbolBundle(outputBundle, opts)

	if cache != nil {
		cache.SetSearch(ctx.Pattern, ctx.CacheKey, resolved.Output, affectedFiles)
		storeSinglePatternBundle(ctx.Pattern, ctx.CacheKey, resolved.Bundle)
		storeSearchAffectedFiles(ctx.Pattern, ctx.CacheKey, affectedFiles)
		storeSinglePatternObservation(ctx.Pattern, ctx.CacheKey, observation)
	}

	return structuredImpactExecutionResult{
		Rendered:      output,
		Bundle:        outputBundle,
		AffectedFiles: affectedFiles,
		Observation:   observation,
	}, true
}

func isValidStructuredImpactSingleBundle(bundle *SymbolBundle) bool {
	return bundle != nil && bundle.Impact != nil && len(bundle.Impact.RecommendedReads) > 0
}

func isStructuredImpactAmbiguousOutput(output string) bool {
	output = strings.TrimSpace(output)
	return strings.HasPrefix(output, "Multiple symbols matched ") || strings.HasPrefix(output, "Multiple definitions found ")
}

func resolveStructuredImpactMultipleResult(cache tools.ToolCacheInterface, ctx structuredImpactSearchContext, opts SearchOptions, resolved symbolResolveResult) structuredImpactExecutionResult {
	affectedFiles := append([]string(nil), resolved.AffectedFiles...)
	if len(affectedFiles) == 0 {
		affectedFiles = deriveAffectedFilesFromCachedResult(nil, resolved.Output, opts)
	}

	if cache != nil {
		cache.SetSearch(ctx.Pattern, ctx.CacheKey, resolved.Output, affectedFiles)
		storeSearchAffectedFiles(ctx.Pattern, ctx.CacheKey, affectedFiles)
		storeSinglePatternBundle(ctx.Pattern, ctx.CacheKey, nil)
		storeSinglePatternObservation(ctx.Pattern, ctx.CacheKey, resolved.Observation)
	}

	return structuredImpactExecutionResult{
		Rendered:      resolved.Output,
		AffectedFiles: affectedFiles,
		Observation:   resolved.Observation,
		Ambiguous:     true,
	}
}

func newStructuredImpactSearchArtifact(result structuredImpactExecutionResult) SearchExecutionArtifact {
	return SearchExecutionArtifact{
		Rendered: result.Rendered,
		Metadata: SearchExecutionMetadata{
			Bundle:           result.Bundle,
			AffectedFiles:    result.AffectedFiles,
			Observation:      tools.CloneRuntimeObservation(result.Observation),
			StructuredImpact: true,
			Ambiguous:        result.Ambiguous,
			MultiPattern:     result.MultiPattern,
		},
	}
}
