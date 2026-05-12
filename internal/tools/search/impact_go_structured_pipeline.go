package search

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

func tryStructuredGoImpactSearchResult(cache tools.ToolCacheInterface, opts SearchOptions) (structuredImpactExecutionResult, bool) {
	ctx, ok := newStructuredGoImpactSearchContext(opts)
	if !ok {
		return structuredImpactExecutionResult{}, false
	}
	return tryStructuredImpactSearchResult(cache, ctx, opts, resolveStructuredGoImpactSymbol)
}

func newStructuredGoImpactSearchContext(opts SearchOptions) (structuredImpactSearchContext, bool) {
	pattern := strings.TrimSpace(opts.Pattern)
	if !shouldAttemptStructuredGoImpactSearch(opts, pattern) {
		return structuredImpactSearchContext{}, false
	}

	ctx, ok := newStructuredImpactSearchContext(opts, structuredGoImpactRouteTag)
	if !ok {
		return structuredImpactSearchContext{}, false
	}
	if ctx.Route.Language != "go" {
		return structuredImpactSearchContext{}, false
	}
	return ctx, true
}
