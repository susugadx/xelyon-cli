package search

import (
	"github.com/susugadx/xelyon-cli/internal/tools"
)

type singlePatternExecutionContext struct {
	Pattern  string
	Opts     SearchOptions
	Route    searchRouteTrace
	CacheKey string
}

func executeSinglePatternWithTrace(cache tools.ToolCacheInterface, pattern string, opts SearchOptions) (string, searchRouteTrace) {
	result := executeSinglePatternDetailed(cache, pattern, opts)
	return result.Output, result.Route
}

func executeSinglePatternDetailed(cache tools.ToolCacheInterface, pattern string, opts SearchOptions) singlePatternExecution {
	return executeSinglePatternWithContext(cache, newSinglePatternExecutionContext(pattern, opts))
}

func executeSinglePatternWithContext(cache tools.ToolCacheInterface, ctx singlePatternExecutionContext) singlePatternExecution {
	if cached, ok := loadCachedSinglePatternExecution(cache, ctx); ok {
		return cached
	}
	if resolved, ok := executeSinglePatternSymbolStage(cache, &ctx); ok {
		return resolved
	}
	return executeSinglePatternTextStage(cache, ctx)
}

func newSinglePatternExecutionContext(pattern string, opts SearchOptions) singlePatternExecutionContext {
	route := planSearchRoute(pattern, opts)
	return singlePatternExecutionContext{
		Pattern:  pattern,
		Opts:     opts,
		Route:    route,
		CacheKey: buildSearchCacheKey(opts, route),
	}
}

func newSinglePatternExecutionContexts(patterns []string, opts SearchOptions) []singlePatternExecutionContext {
	contexts := make([]singlePatternExecutionContext, 0, len(patterns))
	for _, pattern := range patterns {
		contexts = append(contexts, newSinglePatternExecutionContext(pattern, opts))
	}
	return contexts
}

func loadCachedSinglePatternExecution(cache tools.ToolCacheInterface, ctx singlePatternExecutionContext) (singlePatternExecution, bool) {
	cached, ok := loadCachedSinglePatternOutput(cache, ctx)
	if !ok {
		return singlePatternExecution{}, false
	}
	bundle, formatted := loadCachedSinglePatternBundleOutput(ctx, cache, cached)
	affectedFiles := loadCachedSinglePatternAffectedFiles(ctx, bundle, formatted)
	observation := loadCachedSinglePatternObservation(ctx, bundle)
	return newCachedSinglePatternExecution(ctx, formatted, bundle, affectedFiles, observation), true
}

func loadCachedSinglePatternOutput(cache tools.ToolCacheInterface, ctx singlePatternExecutionContext) (string, bool) {
	if cache == nil {
		return "", false
	}
	return cache.GetSearch(ctx.Pattern, ctx.CacheKey)
}

func loadCachedSinglePatternBundleOutput(ctx singlePatternExecutionContext, cache tools.ToolCacheInterface, output string) (*SymbolBundle, string) {
	bundle := loadSinglePatternBundle(ctx.Pattern, ctx.CacheKey)
	return formatImpactBundleForRuntimeWithContext(bundle, output, ctx.Opts, cache, currentSearchImpactRuntimeRankContext(ctx.Pattern, ctx.CacheKey))
}

func loadCachedSinglePatternAffectedFiles(ctx singlePatternExecutionContext, bundle *SymbolBundle, output string) []string {
	if affectedFiles := loadSearchAffectedFiles(ctx.Pattern, ctx.CacheKey); len(affectedFiles) > 0 {
		return affectedFiles
	}
	return deriveAffectedFilesFromCachedResult(bundle, output, ctx.Opts)
}

func loadCachedSinglePatternObservation(ctx singlePatternExecutionContext, bundle *SymbolBundle) *tools.RuntimeObservation {
	if bundle != nil {
		return observationForSymbolBundle(bundle, ctx.Opts)
	}
	if observation := loadSinglePatternObservation(ctx.Pattern, ctx.CacheKey); observation != nil {
		return observation
	}
	return nil
}

func newCachedSinglePatternExecution(ctx singlePatternExecutionContext, output string, bundle *SymbolBundle, affectedFiles []string, observation *tools.RuntimeObservation) singlePatternExecution {
	return singlePatternExecution{
		Pattern:       ctx.Pattern,
		Output:        output,
		Route:         ctx.Route,
		Bundle:        bundle,
		AffectedFiles: affectedFiles,
		Observation:   tools.CloneRuntimeObservation(observation),
	}
}

func writeSinglePatternSearchCache(cache tools.ToolCacheInterface, ctx singlePatternExecutionContext, output string, affectedFiles []string, observation *tools.RuntimeObservation) {
	if cache == nil {
		return
	}
	cache.SetSearch(ctx.Pattern, ctx.CacheKey, output, affectedFiles)
	storeSinglePatternBundle(ctx.Pattern, ctx.CacheKey, nil)
	storeSearchAffectedFiles(ctx.Pattern, ctx.CacheKey, affectedFiles)
	storeSinglePatternObservation(ctx.Pattern, ctx.CacheKey, observation)
}
