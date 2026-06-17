package search

import "github.com/susugadx/xelyon-cli/internal/tools"

type structuredImpactCachedResult struct {
	Output        string
	Bundle        *SymbolBundle
	AffectedFiles []string
	Observation   *tools.RuntimeObservation
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
