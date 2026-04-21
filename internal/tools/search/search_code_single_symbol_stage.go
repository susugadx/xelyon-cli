package search

import "github.com/susugadx/xelyon-cli/internal/tools"

func executeSinglePatternSymbolStage(cache tools.ToolCacheInterface, ctx *singlePatternExecutionContext) (singlePatternExecution, bool) {
	if ctx.Route.InitialLane != searchLaneSymbol {
		return singlePatternExecution{}, false
	}

	ctx.Route.SymbolAttempted = true
	resolver := resolverForLanguage(ctx.Route.Language)
	if resolver != nil {
		resolved := resolver.Resolve(ctx.Route.SymbolQuery, ctx.Opts)
		switch resolved.Status {
		case symbolResolveSingle:
			return completeSingleSymbolPattern(cache, *ctx, resolved), true
		case symbolResolveMultiple:
			return completeMultiSymbolPattern(cache, *ctx, resolved), true
		case symbolResolveNone:
			ctx.Route.SymbolResolved = false
		}
	}

	if ctx.Route.FallbackLane != "" {
		ctx.Route.FallbackUsed = true
		ctx.Route.FinalLane = ctx.Route.FallbackLane
	}
	return singlePatternExecution{}, false
}

func completeSingleSymbolPattern(cache tools.ToolCacheInterface, ctx singlePatternExecutionContext, resolved symbolResolveResult) singlePatternExecution {
	route := ctx.Route
	route.FinalLane = searchLaneSymbol
	route.SymbolResolved = true
	resolved.Bundle = attachBundleRoute(resolved.Bundle, route)
	outputBundle, output := formatImpactBundleForRuntime(resolved.Bundle, resolved.Output, ctx.Opts, cache)
	affectedFiles := collectSymbolBundleAffectedFiles(resolved.Bundle, ctx.Opts)

	writeSinglePatternSearchCache(cache, ctx, resolved.Output, affectedFiles)
	if cache != nil {
		storeSinglePatternBundle(ctx.Pattern, ctx.CacheKey, resolved.Bundle)
	}

	return singlePatternExecution{
		Pattern:       ctx.Pattern,
		Output:        output,
		Route:         route,
		Bundle:        outputBundle,
		AffectedFiles: affectedFiles,
	}
}

func completeMultiSymbolPattern(cache tools.ToolCacheInterface, ctx singlePatternExecutionContext, resolved symbolResolveResult) singlePatternExecution {
	route := ctx.Route
	route.FinalLane = searchLaneSymbol
	route.SymbolResolved = true

	affectedFiles := append([]string(nil), resolved.AffectedFiles...)
	affectedFiles = append(affectedFiles, collectPrimaryAffectedFilePathsFromOutput(resolved.Output, ctx.Opts)...)
	affectedFiles = dedupePaths(affectedFiles)
	if len(affectedFiles) == 0 {
		affectedFiles = deriveAffectedFilesFromCachedResult(nil, resolved.Output, ctx.Opts)
	}
	writeSinglePatternSearchCache(cache, ctx, resolved.Output, affectedFiles)

	return singlePatternExecution{
		Pattern:       ctx.Pattern,
		Output:        resolved.Output,
		Route:         route,
		AffectedFiles: affectedFiles,
	}
}
