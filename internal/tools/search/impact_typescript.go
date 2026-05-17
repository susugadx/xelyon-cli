package search

import "strings"

const structuredTypeScriptImpactRouteTag = "impact-structured-typescript-v1"

func newStructuredTypeScriptImpactSearchContext(opts SearchOptions) (structuredImpactSearchContext, structuredImpactScope, bool) {
	return structuredTypeScriptImpactLanguageSpec().newSearchContext(opts)
}

func structuredTypeScriptImpactRoute(pattern string, opts SearchOptions) (searchRouteTrace, bool) {
	return structuredImpactSymbolRoute(pattern, opts, "js", structuredTypeScriptImpactRouteTag)
}

func structuredTypeScriptImpactLanguageSpec() structuredImpactLanguageSpec {
	return structuredImpactLanguageSpec{
		name:               "typescript",
		routeTag:           structuredTypeScriptImpactRouteTag,
		normalize:          normalizeStructuredTypeScriptImpactScope,
		planRoute:          structuredTypeScriptImpactRoute,
		resolver:           resolveStructuredTypeScriptImpactSymbol,
		expandSupplemental: true,
	}
}

func normalizeStructuredTypeScriptImpactOptions(opts SearchOptions) (SearchOptions, bool) {
	fileType := strings.ToLower(strings.TrimSpace(opts.FileType))
	filePattern := cleanStructuredTypeScriptFilePattern(opts.FilePattern)

	if fileType != "" {
		if !structuredTypeScriptImpactAllowsFileType(fileType) {
			return SearchOptions{}, false
		}
		opts.FileType = fileType
		opts.FilePattern = ""
		return opts, true
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

func normalizeStructuredTypeScriptImpactScope(opts SearchOptions) (structuredImpactScope, bool) {
	return normalizeStructuredImpactSameScope(opts, normalizeStructuredTypeScriptImpactOptions)
}

func resolveStructuredTypeScriptImpactSymbol(symbol string, scope structuredImpactScope) symbolResolveResult {
	return resolveStructuredJSFamilyImpactSymbol(symbol, scope, structuredTypeScriptImpactResolverSpec())
}

func structuredTypeScriptImpactResolverSpec() jsFamilyImpactResolverSpec {
	return jsFamilyImpactResolverSpec{
		findDefinitions:         findStructuredTypeScriptImpactDefinitionSet,
		collectDefAffectedFiles: collectStructuredTypeScriptDefAffectedFiles,
		referenceOptions:        structuredTypeScriptImpactReferenceOptions,
		normalizeRefs:           normalizeStructuredTypeScriptRefs,
		filterRefs:              filterStructuredTypeScriptImpactRefs,
		buildBundle:             buildTypeScriptImpactBundleFromRefs,
	}
}
