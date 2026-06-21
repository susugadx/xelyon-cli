package search

import "fmt"

const (
	jsImportLimit  = 5
	jsCallerLimit  = 10
	jsTypeRefLimit = 5
)

// resolveJSFamilySymbol は TS/JS 向けの enhanced symbol fast path。
// 参照を import / caller / type ref / other に分類し、Go fast path に近い構造化結果を返す。
// フォールバック: 定義が見つからなければ genericSymbolNone を返し、text search に委譲する。
func resolveJSFamilySymbol(symbol string, opts SearchOptions) genericResolveResult {
	candidates := collectJSFamilyDefinitionCandidates(symbol, opts)
	defs := candidates.astDefs
	if len(defs) == 0 {
		defs = genericDefinitionsFromMatches(symbol, opts, candidates.matches)
	}
	if len(defs) == 0 {
		return genericResolveResult{Status: genericSymbolNone}
	}
	if shouldDeferIncompleteJSFamilyDefinitions(candidates.definitionIncomplete) {
		return genericResolveResult{Status: genericSymbolNone}
	}
	if len(defs) > 1 {
		return genericResolveResult{
			Output:        formatGenericMultipleDefsWithOptions(symbol, defs, opts.LocatorRegistry, opts),
			Status:        genericSymbolMultiple,
			AffectedFiles: collectGenericDefAffectedFiles(defs, opts),
		}
	}

	def := defs[0]
	refResult := findJSFamilyReferencesWithSemantic(symbol, def, newJSFamilyReferenceOptions(opts))
	refs := refResult.refs
	totalRefs := refResult.refsForTotals()
	filteredRefs := filterGenericRefs(refs, def)
	filteredTotalRefs := filterGenericRefs(totalRefs, def)
	classifiedRefs := classifyJSFamilySymbolRefsFromAST(filteredRefs)
	classifiedTotalRefs := classifyJSFamilySymbolRefsFromAST(filteredTotalRefs)
	bundle := buildGenericSymbolBundle("js", symbol, def, []string{
		fmt.Sprintf("%d: %s", def.Line, def.Signature),
	}, []symbolBundleSectionInput{
		{Kind: "imports", Title: "Imports", Items: classifiedRefs.imports, TotalItems: classifiedTotalRefs.imports, Limit: jsImportLimit},
		{Kind: "callers", Title: "Callers", Items: classifiedRefs.callers, TotalItems: classifiedTotalRefs.callers, Limit: jsCallerLimit},
		{Kind: "type_refs", Title: "Type References", Items: classifiedRefs.typeRefs, TotalItems: classifiedTotalRefs.typeRefs, Limit: jsTypeRefLimit},
		{Kind: "references", Title: "References", Items: classifiedRefs.others, TotalItems: classifiedTotalRefs.others, Limit: genericRefLimit},
		{Kind: "tests", Title: "Related Tests", Items: classifiedRefs.tests, TotalItems: classifiedTotalRefs.tests, Limit: genericTestLimit, IsTest: true},
	})
	setJSFamilyBundleDiagnostics(bundle, refResult.diagnostics, filteredTotalRefs)
	bundle.Debug.FileRootPath = invocationCWDOrGetwd(opts)
	return genericResolveResult{Output: formatSymbolBundle(bundle, opts.LocatorRegistry, nil), Status: genericSymbolSingle, Bundle: bundle}
}

type jsFamilySymbolRefs struct {
	imports  []genericSymbolRef
	callers  []genericSymbolRef
	typeRefs []genericSymbolRef
	others   []genericSymbolRef
	tests    []genericSymbolRef
}
