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
	route := resolvedSinglePatternSymbolRoute(ctx.Route)
	resolved.Bundle = attachBundleRoute(resolved.Bundle, route)
	outputBundle, output := formatImpactBundleForRuntime(resolved.Bundle, resolved.Output, ctx.Opts, cache)
	affectedFiles := collectSymbolBundleAffectedFiles(resolved.Bundle, ctx.Opts)

	writeSingleSymbolPatternCache(cache, ctx, resolved, affectedFiles)

	observation := observationForSymbolBundle(outputBundle, ctx.Opts)
	return newSinglePatternSymbolExecution(ctx.Pattern, output, route, outputBundle, affectedFiles, observation)
}

func completeMultiSymbolPattern(cache tools.ToolCacheInterface, ctx singlePatternExecutionContext, resolved symbolResolveResult) singlePatternExecution {
	route := resolvedSinglePatternSymbolRoute(ctx.Route)
	affectedFiles := resolveMultiSymbolAffectedFiles(resolved, ctx.Opts)
	writeSinglePatternSearchCache(cache, ctx, resolved.Output, affectedFiles, resolved.Observation)

	return newSinglePatternSymbolExecution(ctx.Pattern, resolved.Output, route, nil, affectedFiles, resolved.Observation)
}

func resolvedSinglePatternSymbolRoute(route searchRouteTrace) searchRouteTrace {
	route.FinalLane = searchLaneSymbol
	route.SymbolResolved = true
	return route
}

func newSinglePatternSymbolExecution(pattern string, output string, route searchRouteTrace, bundle *SymbolBundle, affectedFiles []string, observation *tools.RuntimeObservation) singlePatternExecution {
	return singlePatternExecution{
		Pattern:       pattern,
		Output:        output,
		Route:         route,
		Bundle:        bundle,
		AffectedFiles: affectedFiles,
		Observation:   tools.CloneRuntimeObservation(observation),
	}
}
