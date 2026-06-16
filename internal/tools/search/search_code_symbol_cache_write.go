package search

import "github.com/susugadx/xelyon-cli/internal/tools"

func writeSingleSymbolPatternCache(cache tools.ToolCacheInterface, ctx singlePatternExecutionContext, resolved symbolResolveResult, affectedFiles []string) {
	observation := resolved.Observation
	if observation == nil {
		observation = observationForSymbolBundle(resolved.Bundle, ctx.Opts)
	}
	writeSinglePatternSearchCache(cache, ctx, resolved.Output, affectedFiles, observation)
	if cache == nil || resolved.Bundle == nil {
		return
	}
	storeSinglePatternBundle(ctx.Pattern, ctx.CacheKey, resolved.Bundle)
}
