package search

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

type structuredImpactExecutionResult struct {
	Rendered      string
	Bundle        *SymbolBundle
	AffectedFiles []string
	Observation   *tools.RuntimeObservation
	Ambiguous     bool
	MultiPattern  bool
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
			Diagnostics:      cloneBundleDiagnosticsForMetadata(result.Bundle),
			AffectedFiles:    result.AffectedFiles,
			Observation:      tools.CloneRuntimeObservation(result.Observation),
			StructuredImpact: true,
			Ambiguous:        result.Ambiguous,
			MultiPattern:     result.MultiPattern,
		},
	}
}
