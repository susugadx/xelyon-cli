package search

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

type structuredGoImpactSearchContext struct {
	Pattern  string
	Route    searchRouteTrace
	CacheKey string
}

type structuredGoImpactCachedResult struct {
	Output        string
	Bundle        *SymbolBundle
	AffectedFiles []string
	Observation   *tools.RuntimeObservation
}

type structuredGoImpactExecutionResult struct {
	Rendered      string
	Bundle        *SymbolBundle
	AffectedFiles []string
	Observation   *tools.RuntimeObservation
	Ambiguous     bool
}

func newStructuredGoImpactSearchContext(opts SearchOptions) (structuredGoImpactSearchContext, bool) {
	pattern := strings.TrimSpace(opts.Pattern)
	if !shouldAttemptStructuredGoImpactSearch(opts, pattern) {
		return structuredGoImpactSearchContext{}, false
	}

	route := planSearchRoute(pattern, opts)
	if route.InitialLane != searchLaneSymbol || route.Language != "go" {
		return structuredGoImpactSearchContext{}, false
	}

	return structuredGoImpactSearchContext{
		Pattern:  pattern,
		Route:    route,
		CacheKey: buildStructuredGoImpactCacheKey(opts, route),
	}, true
}

func buildStructuredGoImpactCacheKey(opts SearchOptions, route searchRouteTrace) string {
	return buildSearchCacheKeyWithRoute(opts, route.cacheSignature()+"|"+structuredGoImpactRouteTag)
}

func loadStructuredGoImpactCachedResult(cache tools.ToolCacheInterface, ctx structuredGoImpactSearchContext, opts SearchOptions) (structuredGoImpactCachedResult, bool) {
	if cache == nil {
		return structuredGoImpactCachedResult{}, false
	}
	cached, ok := cache.GetSearch(ctx.Pattern, ctx.CacheKey)
	if !ok {
		return structuredGoImpactCachedResult{}, false
	}

	bundle := loadSinglePatternBundle(ctx.Pattern, ctx.CacheKey)
	bundle, cached = formatImpactBundleForRuntimeWithContext(bundle, cached, opts, cache, currentSearchImpactRuntimeRankContext(ctx.Pattern, ctx.CacheKey))
	affectedFiles := loadSinglePatternAffectedFiles(ctx.Pattern, ctx.CacheKey)
	if len(affectedFiles) == 0 {
		affectedFiles = deriveAffectedFilesFromCachedResult(bundle, cached, opts)
	}

	return structuredGoImpactCachedResult{
		Output:        cached,
		Bundle:        bundle,
		AffectedFiles: affectedFiles,
		Observation:   loadCachedStructuredGoImpactObservation(ctx, bundle, opts),
	}, true
}

func loadCachedStructuredGoImpactObservation(ctx structuredGoImpactSearchContext, bundle *SymbolBundle, opts SearchOptions) *tools.RuntimeObservation {
	if observation := loadSinglePatternObservation(ctx.Pattern, ctx.CacheKey); observation != nil {
		return observation
	}
	if bundle != nil {
		return observationForSymbolBundle(bundle, opts)
	}
	return nil
}

func resolveStructuredGoImpactWithContext(cache tools.ToolCacheInterface, ctx structuredGoImpactSearchContext, opts SearchOptions) (structuredGoImpactExecutionResult, bool) {
	resolved := resolveStructuredGoImpactSymbol(ctx.Pattern, opts)
	route := ctx.Route
	route.SymbolAttempted = true

	switch resolved.Status {
	case symbolResolveSingle:
		return resolveStructuredGoImpactSingleResult(cache, ctx, opts, route, resolved)
	case symbolResolveMultiple:
		return resolveStructuredGoImpactMultipleResult(cache, ctx, opts, resolved), true
	default:
		return structuredGoImpactExecutionResult{}, false
	}
}

func resolveStructuredGoImpactSingleResult(cache tools.ToolCacheInterface, ctx structuredGoImpactSearchContext, opts SearchOptions, route searchRouteTrace, resolved symbolResolveResult) (structuredGoImpactExecutionResult, bool) {
	if resolved.Bundle == nil {
		return structuredGoImpactExecutionResult{}, false
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
		storeSinglePatternAffectedFiles(ctx.Pattern, ctx.CacheKey, affectedFiles)
		storeSinglePatternObservation(ctx.Pattern, ctx.CacheKey, observation)
	}

	return structuredGoImpactExecutionResult{
		Rendered:      output,
		Bundle:        outputBundle,
		AffectedFiles: affectedFiles,
		Observation:   observation,
	}, true
}

func resolveStructuredGoImpactMultipleResult(cache tools.ToolCacheInterface, ctx structuredGoImpactSearchContext, opts SearchOptions, resolved symbolResolveResult) structuredGoImpactExecutionResult {
	affectedFiles := append([]string(nil), resolved.AffectedFiles...)
	if len(affectedFiles) == 0 {
		affectedFiles = deriveAffectedFilesFromCachedResult(nil, resolved.Output, opts)
	}

	if cache != nil {
		cache.SetSearch(ctx.Pattern, ctx.CacheKey, resolved.Output, affectedFiles)
		storeSinglePatternAffectedFiles(ctx.Pattern, ctx.CacheKey, affectedFiles)
		storeSinglePatternObservation(ctx.Pattern, ctx.CacheKey, resolved.Observation)
	}

	return structuredGoImpactExecutionResult{
		Rendered:      resolved.Output,
		AffectedFiles: affectedFiles,
		Observation:   resolved.Observation,
		Ambiguous:     true,
	}
}
