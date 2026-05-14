package search

import (
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/locator"
)

const (
	jsImportLimit  = 5
	jsCallerLimit  = 10
	jsTypeRefLimit = 5
)

// resolveJSSymbol は TS/JS 向けの enhanced symbol fast path。
// 参照を import / caller / type ref / other に分類し、Go fast path に近い構造化結果を返す。
// フォールバック: 定義が見つからなければ genericSymbolNone を返し、text search に委譲する。
func resolveJSSymbol(symbol string, opts SearchOptions) genericResolveResult {
	defs := findJSFamilyDefinitionsWithAST(symbol, opts)
	if len(defs) == 0 {
		defs = findGenericDefinitions(symbol, opts)
	}
	if len(defs) == 0 {
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
	filteredRefs := filterGenericRefs(refs, def)
	classifiedRefs := classifyJSFamilySymbolRefsFromAST(filteredRefs)
	bundle := buildGenericSymbolBundle("js", symbol, def, []string{
		fmt.Sprintf("%d: %s", def.Line, def.Signature),
	}, []symbolBundleSectionInput{
		{Kind: "imports", Title: "Imports", Items: classifiedRefs.imports, Limit: jsImportLimit},
		{Kind: "callers", Title: "Callers", Items: classifiedRefs.callers, Limit: jsCallerLimit},
		{Kind: "type_refs", Title: "Type References", Items: classifiedRefs.typeRefs, Limit: jsTypeRefLimit},
		{Kind: "references", Title: "References", Items: classifiedRefs.others, Limit: genericRefLimit},
		{Kind: "tests", Title: "Related Tests", Items: classifiedRefs.tests, Limit: genericTestLimit, IsTest: true},
	})
	setJSFamilyBundleLSPDiagnostics(bundle, refResult.resolvedViaLSP)
	bundle.Debug.FileRootPath = invocationCWDOrGetwd(opts)
	return genericResolveResult{Output: formatJSSymbolResult(bundle, opts.LocatorRegistry), Status: genericSymbolSingle, Bundle: bundle}
}

type jsFamilySymbolRefs struct {
	imports  []genericSymbolRef
	callers  []genericSymbolRef
	typeRefs []genericSymbolRef
	others   []genericSymbolRef
	tests    []genericSymbolRef
}

// formatJSSymbolResult は TS/JS の分類済みシンボル結果をフォーマットする。
// 出力形式は Go fast path と同様のセクション構造で、locator ID を付与する。
func formatJSSymbolResult(bundle *SymbolBundle, reg *locator.Registry) string {
	return formatSymbolBundle(bundle, reg, nil)
}
