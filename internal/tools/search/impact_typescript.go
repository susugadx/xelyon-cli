package search

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

const structuredTypeScriptImpactRouteTag = "impact-structured-typescript-v1"

func tryExpandedStructuredTypeScriptImpactSearchResult(cache tools.ToolCacheInterface, opts SearchOptions) (structuredImpactExecutionResult, bool) {
	result, ok := tryStructuredTypeScriptImpactSearchResult(cache, opts)
	if !ok {
		return structuredImpactExecutionResult{}, false
	}
	return expandStructuredImpactSearchResult(cache, opts, result), true
}

func tryStructuredTypeScriptImpactSearchResult(cache tools.ToolCacheInterface, opts SearchOptions) (structuredImpactExecutionResult, bool) {
	ctx, scope, ok := newStructuredTypeScriptImpactSearchContext(opts)
	if !ok {
		return structuredImpactExecutionResult{}, false
	}
	return tryStructuredImpactSearchResult(cache, ctx, scope, resolveStructuredTypeScriptImpactSymbol)
}

func newStructuredTypeScriptImpactSearchContext(opts SearchOptions) (structuredImpactSearchContext, structuredImpactScope, bool) {
	return newStructuredImpactSearchContext(opts, structuredTypeScriptImpactRouteTag, normalizeStructuredTypeScriptImpactScope, structuredTypeScriptImpactRoute)
}

func structuredTypeScriptImpactRoute(pattern string, opts SearchOptions) (searchRouteTrace, bool) {
	return structuredImpactSymbolRoute(pattern, opts, "js", structuredTypeScriptImpactRouteTag)
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
	opts := scope.Definition
	evidenceOpts := scope.Evidence
	defs := normalizeStructuredTypeScriptDefs(findJSFamilyDefinitionsWithAST(symbol, opts))
	if len(defs) == 0 {
		defs = normalizeStructuredTypeScriptDefs(findGenericDefinitions(symbol, opts))
	}
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
	refOpts := structuredTypeScriptImpactReferenceOptions(def, evidenceOpts)
	refResult := findJSFamilyReferencesWithSemantic(symbol, def, refOpts)
	refs := normalizeStructuredTypeScriptRefs(refResult.refs)
	refs = filterStructuredTypeScriptSuppressedDeclarationRefs(refs, structuredTypeScriptSuppressedDeclarationDefsForImpact(def, preferredDefs))
	filteredRefs := filterGenericRefs(refs, def)
	classifiedRefs := typeScriptImpactRefsForDef(def, filteredRefs, refOpts.nameOnly)
	bundle := buildTypeScriptImpactBundle(symbol, def, refOpts.nameOnly, classifiedRefs)
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
