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

func executeSinglePattern(cache tools.ToolCacheInterface, pattern string, opts SearchOptions) string {
	return executeSinglePatternDetailed(cache, pattern, opts).Output
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
	return newCachedSinglePatternExecution(ctx, formatted, bundle, affectedFiles), true
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
	if affectedFiles := loadSinglePatternAffectedFiles(ctx.Pattern, ctx.CacheKey); len(affectedFiles) > 0 {
		return affectedFiles
	}
	return deriveAffectedFilesFromCachedResult(bundle, output, ctx.Opts)
}

func newCachedSinglePatternExecution(ctx singlePatternExecutionContext, output string, bundle *SymbolBundle, affectedFiles []string) singlePatternExecution {
	return singlePatternExecution{
		Pattern:       ctx.Pattern,
		Output:        output,
		Route:         ctx.Route,
		Bundle:        bundle,
		AffectedFiles: affectedFiles,
	}
}

func writeSinglePatternSearchCache(cache tools.ToolCacheInterface, ctx singlePatternExecutionContext, output string, affectedFiles []string) {
	if cache == nil {
		return
	}
	cache.SetSearch(ctx.Pattern, ctx.CacheKey, output, affectedFiles)
	storeSinglePatternAffectedFiles(ctx.Pattern, ctx.CacheKey, affectedFiles)
}
