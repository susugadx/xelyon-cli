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

func normalizeStructuredJavaScriptImpactScope(opts SearchOptions) (structuredImpactScope, bool) {
	return normalizeStructuredImpactSameScope(opts, normalizeStructuredJavaScriptImpactOptions)
}

func resolveStructuredJavaScriptImpactSymbol(symbol string, scope structuredImpactScope) symbolResolveResult {
	opts := scope.Definition
	evidenceOpts := scope.Evidence
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
	refOpts := structuredJavaScriptImpactReferenceOptions(def, evidenceOpts)
	refResult := findJSFamilyReferencesWithSemantic(symbol, def, refOpts)
	refs := normalizeStructuredJavaScriptRefs(refResult.refs)
	refs = filterGenericRefs(refs, def)
	classifiedRefs := javaScriptImpactRefsForDef(def, refs, refOpts.nameOnly)
	bundle := buildJavaScriptImpactBundle(symbol, def, refOpts.nameOnly, classifiedRefs)
	if bundle == nil || bundle.Impact == nil || len(bundle.Impact.RecommendedReads) == 0 {
		return symbolResolveResult{Status: symbolResolveNone}
	}
	setJSFamilyBundleLSPDiagnostics(bundle, refResult.resolvedViaLSP)

	return symbolResolveResult{
		Output: formatSymbolBundle(bundle, opts.LocatorRegistry, nil),
		Status: symbolResolveSingle,
		Bundle: bundle,
	}
}
