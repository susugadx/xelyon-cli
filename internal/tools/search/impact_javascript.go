package search

import "strings"

const structuredJavaScriptImpactRouteTag = "impact-structured-javascript-v1"

func newStructuredJavaScriptImpactSearchContext(opts SearchOptions) (structuredImpactSearchContext, structuredImpactScope, bool) {
	return structuredJavaScriptImpactLanguageSpec().newSearchContext(opts)
}

func structuredJavaScriptImpactRoute(pattern string, opts SearchOptions) (searchRouteTrace, bool) {
	return structuredImpactSymbolRoute(pattern, opts, "js", structuredJavaScriptImpactRouteTag)
}

func structuredJavaScriptImpactLanguageSpec() structuredImpactLanguageSpec {
	return structuredImpactLanguageSpec{
		name:               "javascript",
		routeTag:           structuredJavaScriptImpactRouteTag,
		normalize:          normalizeStructuredJavaScriptImpactScope,
		planRoute:          structuredJavaScriptImpactRoute,
		resolver:           resolveStructuredJavaScriptImpactSymbol,
		expandSupplemental: true,
	}
}

func normalizeStructuredJavaScriptImpactOptions(opts SearchOptions) (SearchOptions, bool) {
	fileType := strings.ToLower(strings.TrimSpace(opts.FileType))
	filePattern := cleanStructuredJavaScriptFilePattern(opts.FilePattern)

	if fileType != "" {
		target, ok := structuredJavaScriptImpactTargetForFileType(fileType)
		if !ok {
			return SearchOptions{}, false
		}
		opts.FileType = target.fileType
		opts.FilePattern = ""
		return opts, true
	}

	switch {
	case structuredJavaScriptImpactAllowsFilePattern(filePattern):
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

func normalizeStructuredJavaScriptImpactScope(opts SearchOptions) (structuredImpactScope, bool) {
	return normalizeStructuredImpactSameScope(opts, normalizeStructuredJavaScriptImpactOptions)
}

func resolveStructuredJavaScriptImpactSymbol(symbol string, scope structuredImpactScope) symbolResolveResult {
	return resolveStructuredJSFamilyImpactSymbol(symbol, scope, structuredJavaScriptImpactResolverSpec())
}

func structuredJavaScriptImpactResolverSpec() jsFamilyImpactResolverSpec {
	return jsFamilyImpactResolverSpec{
		findDefinitions:         findStructuredJavaScriptImpactDefinitionSet,
		collectDefAffectedFiles: collectStructuredJavaScriptDefAffectedFiles,
		referenceOptions:        structuredJavaScriptImpactReferenceOptions,
		normalizeRefs:           normalizeStructuredJavaScriptRefs,
		language:                "javascript",
		rootPath:                structuredJavaScriptImpactFileRoot,
		debugSource: func(genericSymbolDef) string {
			return "javascript-impact-structured"
		},
		buildSemanticEvidence: buildJSFamilySemanticEvidence,
	}
}
