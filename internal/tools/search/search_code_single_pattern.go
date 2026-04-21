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
	ctx := newSinglePatternExecutionContext(pattern, opts)
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

func loadCachedSinglePatternExecution(cache tools.ToolCacheInterface, ctx singlePatternExecutionContext) (singlePatternExecution, bool) {
	if cache == nil {
		return singlePatternExecution{}, false
	}

	cached, ok := cache.GetSearch(ctx.Pattern, ctx.CacheKey)
	if !ok {
		return singlePatternExecution{}, false
	}

	bundle := loadSinglePatternBundle(ctx.Pattern, ctx.CacheKey)
	bundle, cached = formatImpactBundleForRuntimeWithContext(bundle, cached, ctx.Opts, cache, currentSearchImpactRuntimeRankContext(ctx.Pattern, ctx.CacheKey))
	affectedFiles := loadSinglePatternAffectedFiles(ctx.Pattern, ctx.CacheKey)
	if len(affectedFiles) == 0 {
		affectedFiles = deriveAffectedFilesFromCachedResult(bundle, cached, ctx.Opts)
	}

	return singlePatternExecution{
		Pattern:       ctx.Pattern,
		Output:        cached,
		Route:         ctx.Route,
		Bundle:        bundle,
		AffectedFiles: affectedFiles,
	}, true
}

func writeSinglePatternSearchCache(cache tools.ToolCacheInterface, ctx singlePatternExecutionContext, output string, affectedFiles []string) {
	if cache == nil {
		return
	}
	cache.SetSearch(ctx.Pattern, ctx.CacheKey, output, affectedFiles)
	storeSinglePatternAffectedFiles(ctx.Pattern, ctx.CacheKey, affectedFiles)
}
