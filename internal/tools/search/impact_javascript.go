package search

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

const structuredJavaScriptImpactRouteTag = "impact-structured-javascript-v1"

func tryExpandedStructuredJavaScriptImpactSearchResult(cache tools.ToolCacheInterface, opts SearchOptions) (structuredImpactExecutionResult, bool) {
	result, ok := tryStructuredJavaScriptImpactSearchResult(cache, opts)
	if !ok {
		return structuredImpactExecutionResult{}, false
	}
	return expandStructuredImpactSearchResult(cache, opts, result), true
}

func tryStructuredJavaScriptImpactSearchResult(cache tools.ToolCacheInterface, opts SearchOptions) (structuredImpactExecutionResult, bool) {
	ctx, resolverOpts, ok := newStructuredJavaScriptImpactSearchContext(opts)
	if !ok {
		return structuredImpactExecutionResult{}, false
	}
	return tryStructuredImpactSearchResult(cache, ctx, resolverOpts, resolveStructuredJavaScriptImpactSymbol)
}

func newStructuredJavaScriptImpactSearchContext(opts SearchOptions) (structuredImpactSearchContext, SearchOptions, bool) {
	pattern := strings.TrimSpace(opts.Pattern)
	resolverOpts, ok := normalizeStructuredJavaScriptImpactOptions(opts)
	if !shouldAttemptStructuredJavaScriptImpactSearch(opts, pattern) || !ok {
		return structuredImpactSearchContext{}, SearchOptions{}, false
	}

	route, ok := structuredJavaScriptImpactRoute(pattern, opts)
	if !ok {
		return structuredImpactSearchContext{}, SearchOptions{}, false
	}

	return structuredImpactSearchContext{
		Pattern:  pattern,
		Route:    route,
		CacheKey: buildStructuredImpactCacheKey(opts, route, structuredJavaScriptImpactRouteTag),
	}, resolverOpts, true
}

func shouldAttemptStructuredJavaScriptImpactSearch(opts SearchOptions, pattern string) bool {
	return shouldAttemptSinglePatternImpactSearch(opts, pattern)
}

func structuredJavaScriptImpactRoute(pattern string, opts SearchOptions) (searchRouteTrace, bool) {
	analysis := analyzeSearchQuery(pattern)
	if !analysis.LooksLikeBareIdentifier && !analysis.LooksLikeDottedSymbol {
		return searchRouteTrace{}, false
	}

	route := planSearchRoute(pattern, opts)
	if route.InitialLane == searchLaneSymbol {
		if route.Language == "" {
			route.Language = "js"
		}
		return route, true
	}

	mode := SearchMode(opts.Mode)
	if mode == SearchModeLiteral || mode == SearchModeRegex {
		return searchRouteTrace{}, false
	}

	return assignRouteSymbolQuery(searchRouteTrace{
		RequestedMode: mode,
		Language:      "js",
		FallbackLane:  analysis.defaultTextLane(),
		Decision:      structuredJavaScriptImpactRouteTag,
		Analysis:      analysis,
	}, analysis.TrimmedPattern, []string{analysis.TrimmedPattern}), true
}

func normalizeStructuredJavaScriptImpactOptions(opts SearchOptions) (SearchOptions, bool) {
	fileType := strings.ToLower(strings.TrimSpace(opts.FileType))
	filePattern := cleanStructuredJavaScriptFilePattern(opts.FilePattern)

	if fileType != "" {
		if fileType != "js" {
			return SearchOptions{}, false
		}
		opts.FileType = "js"
		opts.FilePattern = ""
		return opts, true
	}

	switch {
	case filePattern == "*.js":
		opts.FileType = ""
		opts.FilePattern = "*.js"
		return opts, true
	case isJavaScriptOnlyFilePattern(filePattern):
		opts.FileType = ""
		opts.FilePattern = filePattern
		return opts, true
	case filePattern != "":
		return SearchOptions{}, false
	case isJavaScriptSourceFilePath(opts.Path):
		return opts, true
	default:
		return SearchOptions{}, false
	}
}

func resolveStructuredJavaScriptImpactSymbol(symbol string, opts SearchOptions) symbolResolveResult {
	defs := findStructuredJavaScriptImpactDefinitions(symbol, opts)
	if len(defs) == 0 {
		return symbolResolveResult{Status: symbolResolveNone}
	}
	if len(defs) > 1 {
		return symbolResolveResult{
			Output:        formatGenericMultipleDefsWithOptions(symbol, defs, opts.LocatorRegistry, opts),
			Status:        symbolResolveMultiple,
			AffectedFiles: collectStructuredJavaScriptDefAffectedFiles(defs, opts),
		}
	}

	def := defs[0]
	refResult := findJSFamilyReferencesWithSemantic(symbol, def, opts)
	refs := normalizeStructuredJavaScriptRefs(refResult.refs)
	refs = filterGenericRefs(refs, def)
	classifiedRefs := javaScriptImpactRefsForDef(def, refs, opts)
	bundle := buildJavaScriptImpactBundle(symbol, def, opts, classifiedRefs)
	if bundle == nil || bundle.Impact == nil || len(bundle.Impact.RecommendedReads) == 0 {
		return symbolResolveResult{Status: symbolResolveNone}
	}
	bundle.Diagnostics.ResolvedViaLSP = refResult.resolvedViaLSP

	return symbolResolveResult{
		Output: formatSymbolBundle(bundle, opts.LocatorRegistry, nil),
		Status: symbolResolveSingle,
		Bundle: bundle,
	}
}
