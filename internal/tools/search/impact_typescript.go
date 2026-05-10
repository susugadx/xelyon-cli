package search

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

const structuredTypeScriptImpactRouteTag = "impact-structured-typescript-v1"

func tryStructuredTypeScriptImpactSearch(cache tools.ToolCacheInterface, opts SearchOptions) (string, bool) {
	result, ok := tryExpandedStructuredTypeScriptImpactSearchResult(cache, opts)
	if !ok {
		return "", false
	}
	return result.Rendered, true
}

func tryExpandedStructuredTypeScriptImpactSearchResult(cache tools.ToolCacheInterface, opts SearchOptions) (structuredImpactExecutionResult, bool) {
	result, ok := tryStructuredTypeScriptImpactSearchResult(cache, opts)
	if !ok {
		return structuredImpactExecutionResult{}, false
	}
	return expandStructuredImpactSearchResult(cache, opts, result), true
}

func tryStructuredTypeScriptImpactSearchResult(cache tools.ToolCacheInterface, opts SearchOptions) (structuredImpactExecutionResult, bool) {
	ctx, resolverOpts, ok := newStructuredTypeScriptImpactSearchContext(opts)
	if !ok {
		return structuredImpactExecutionResult{}, false
	}
	return tryStructuredImpactSearchResult(cache, ctx, resolverOpts, resolveStructuredTypeScriptImpactSymbol)
}

func tryStructuredTypeScriptImpactSearchArtifact(cache tools.ToolCacheInterface, opts SearchOptions) (SearchExecutionArtifact, bool) {
	result, ok := tryExpandedStructuredTypeScriptImpactSearchResult(cache, opts)
	if !ok {
		return SearchExecutionArtifact{}, false
	}
	return newStructuredImpactSearchArtifact(result), true
}

func newStructuredTypeScriptImpactSearchContext(opts SearchOptions) (structuredImpactSearchContext, SearchOptions, bool) {
	pattern := strings.TrimSpace(opts.Pattern)
	resolverOpts, ok := normalizeStructuredTypeScriptImpactOptions(opts)
	if !shouldAttemptStructuredTypeScriptImpactSearch(opts, pattern) || !ok {
		return structuredImpactSearchContext{}, SearchOptions{}, false
	}

	route, ok := structuredTypeScriptImpactRoute(pattern, opts)
	if !ok {
		return structuredImpactSearchContext{}, SearchOptions{}, false
	}

	return structuredImpactSearchContext{
		Pattern:  pattern,
		Route:    route,
		CacheKey: buildStructuredImpactCacheKey(opts, route, structuredTypeScriptImpactRouteTag),
	}, resolverOpts, true
}

func shouldAttemptStructuredTypeScriptImpactSearch(opts SearchOptions, pattern string) bool {
	return shouldAttemptSinglePatternImpactSearch(opts, pattern)
}

func structuredTypeScriptImpactRoute(pattern string, opts SearchOptions) (searchRouteTrace, bool) {
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
		Decision:      structuredTypeScriptImpactRouteTag,
		Analysis:      analysis,
	}, analysis.TrimmedPattern, []string{analysis.TrimmedPattern}), true
}

func normalizeStructuredTypeScriptImpactOptions(opts SearchOptions) (SearchOptions, bool) {
	fileType := strings.ToLower(strings.TrimSpace(opts.FileType))
	filePattern := cleanStructuredTypeScriptFilePattern(opts.FilePattern)

	switch fileType {
	case "ts":
		opts.FileType = "ts"
		opts.FilePattern = ""
		return opts, true
	case "typescript", "tsx", "js", "jsx", "mjs", "cjs", "javascript":
		return SearchOptions{}, false
	case "":
	default:
		return SearchOptions{}, false
	}

	switch {
	case filePattern == "*.ts":
		opts.FileType = ""
		opts.FilePattern = "*.ts"
		return opts, true
	case isTypeScriptOnlyFilePattern(filePattern):
		opts.FileType = ""
		opts.FilePattern = filePattern
		return opts, true
	case filePattern != "":
		return SearchOptions{}, false
	case isTypeScriptSourceFilePath(opts.Path):
		return opts, true
	default:
		return SearchOptions{}, false
	}
}

func resolveStructuredTypeScriptImpactSymbol(symbol string, opts SearchOptions) symbolResolveResult {
	defs := normalizeStructuredTypeScriptDefs(findGenericDefinitions(symbol, opts))
	preferredDefs := preferStructuredTypeScriptImplementationDefs(defs)
	defs = preferredDefs.defs
	if len(defs) == 0 {
		return symbolResolveResult{Status: symbolResolveNone}
	}
	if len(defs) > 1 {
		return symbolResolveResult{
			Output:        formatGenericMultipleDefsWithOptions(symbol, defs, opts.LocatorRegistry, opts),
			Status:        symbolResolveMultiple,
			AffectedFiles: collectStructuredTypeScriptDefAffectedFiles(defs, opts),
		}
	}

	def := defs[0]
	refs := normalizeStructuredTypeScriptRefs(findGenericReferences(symbol, opts))
	refs = filterStructuredTypeScriptSuppressedDeclarationRefs(refs, preferredDefs.suppressedDeclarationDefs)
	filteredRefs := filterGenericRefs(refs, def)
	classifiedRefs := typeScriptImpactRefsForDef(def, filteredRefs, opts, symbol)
	bundle := buildTypeScriptImpactBundle(symbol, def, opts, classifiedRefs)
	if bundle == nil || bundle.Impact == nil || len(bundle.Impact.RecommendedReads) == 0 {
		return symbolResolveResult{Status: symbolResolveNone}
	}

	return symbolResolveResult{
		Output: formatSymbolBundle(bundle, opts.LocatorRegistry, nil),
		Status: symbolResolveSingle,
		Bundle: bundle,
	}
}
