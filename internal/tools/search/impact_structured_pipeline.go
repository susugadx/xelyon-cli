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

type structuredImpactCachedResult struct {
	Output        string
	Bundle        *SymbolBundle
	AffectedFiles []string
}

type structuredImpactExecutionResult struct {
	Rendered      string
	Bundle        *SymbolBundle
	AffectedFiles []string
	Ambiguous     bool
	MultiPattern  bool
}

type structuredImpactResolver func(symbol string, opts SearchOptions) symbolResolveResult

func newStructuredImpactSearchContext(opts SearchOptions, routeTag string) (structuredImpactSearchContext, bool) {
	pattern := strings.TrimSpace(opts.Pattern)
	if pattern == "" {
		return structuredImpactSearchContext{}, false
	}

	route := planSearchRoute(pattern, opts)
	if route.InitialLane != searchLaneSymbol {
		return structuredImpactSearchContext{}, false
	}

	return structuredImpactSearchContext{
		Pattern:  pattern,
		Route:    route,
		CacheKey: buildStructuredImpactCacheKey(opts, route, routeTag),
	}, true
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

func tryStructuredImpactSearchResult(cache tools.ToolCacheInterface, ctx structuredImpactSearchContext, opts SearchOptions, resolver structuredImpactResolver) (structuredImpactExecutionResult, bool) {
	if cached, ok := loadStructuredImpactCachedResult(cache, ctx, opts); ok {
		return structuredImpactExecutionResult{
			Rendered:      cached.Output,
			Bundle:        cached.Bundle,
			AffectedFiles: cached.AffectedFiles,
			Ambiguous:     cached.Bundle == nil,
		}, true
	}

	return resolveStructuredImpactWithContext(cache, ctx, opts, resolver)
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
	affectedFiles := loadSinglePatternAffectedFiles(ctx.Pattern, ctx.CacheKey)
	if len(affectedFiles) == 0 {
		affectedFiles = deriveAffectedFilesFromCachedResult(bundle, cached, opts)
	}

	return structuredImpactCachedResult{
		Output:        cached,
		Bundle:        bundle,
		AffectedFiles: affectedFiles,
	}, true
}

func resolveStructuredImpactWithContext(cache tools.ToolCacheInterface, ctx structuredImpactSearchContext, opts SearchOptions, resolver structuredImpactResolver) (structuredImpactExecutionResult, bool) {
	if resolver == nil {
		return structuredImpactExecutionResult{}, false
	}

	resolved := resolver(ctx.Pattern, opts)
	route := ctx.Route
	route.SymbolAttempted = true

	switch resolved.Status {
	case symbolResolveSingle:
		return resolveStructuredImpactSingleResult(cache, ctx, opts, route, resolved)
	case symbolResolveMultiple:
		return resolveStructuredImpactMultipleResult(cache, ctx, opts, resolved), true
	default:
		return structuredImpactExecutionResult{}, false
	}
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

	if cache != nil {
		cache.SetSearch(ctx.Pattern, ctx.CacheKey, resolved.Output, affectedFiles)
		storeSinglePatternBundle(ctx.Pattern, ctx.CacheKey, resolved.Bundle)
		storeSinglePatternAffectedFiles(ctx.Pattern, ctx.CacheKey, affectedFiles)
	}

	return structuredImpactExecutionResult{
		Rendered:      output,
		Bundle:        outputBundle,
		AffectedFiles: affectedFiles,
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
		storeSinglePatternAffectedFiles(ctx.Pattern, ctx.CacheKey, affectedFiles)
	}

	return structuredImpactExecutionResult{
		Rendered:      resolved.Output,
		AffectedFiles: affectedFiles,
		Ambiguous:     true,
	}
}

func newStructuredImpactSearchArtifact(result structuredImpactExecutionResult) SearchExecutionArtifact {
	return SearchExecutionArtifact{
		Rendered: result.Rendered,
		Metadata: SearchExecutionMetadata{
			Bundle:           result.Bundle,
			AffectedFiles:    result.AffectedFiles,
			StructuredImpact: true,
			Ambiguous:        result.Ambiguous,
			MultiPattern:     result.MultiPattern,
		},
	}
}
