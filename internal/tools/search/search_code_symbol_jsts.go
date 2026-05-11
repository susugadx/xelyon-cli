package search

import (
	"fmt"
	"regexp"

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
	defs := findGenericDefinitions(symbol, opts)
	if len(defs) == 0 {
		return genericResolveResult{Status: genericSymbolNone}
	}
	if len(defs) > 1 {
		return genericResolveResult{Output: formatGenericMultipleDefsWithOptions(symbol, defs, opts.LocatorRegistry, opts), Status: genericSymbolMultiple}
	}

	def := defs[0]
	refs := findGenericReferences(symbol, opts)
	filteredRefs := filterGenericRefs(refs, def)
	classifiedRefs := classifyJSFamilySymbolRefs(filteredRefs, symbol)
	bundle := buildGenericSymbolBundle("js", symbol, def, []string{
		fmt.Sprintf("%d: %s", def.Line, def.Signature),
	}, []symbolBundleSectionInput{
		{Kind: "imports", Title: "Imports", Items: classifiedRefs.imports, Limit: jsImportLimit},
		{Kind: "callers", Title: "Callers", Items: classifiedRefs.callers, Limit: jsCallerLimit},
		{Kind: "type_refs", Title: "Type References", Items: classifiedRefs.typeRefs, Limit: jsTypeRefLimit},
		{Kind: "references", Title: "References", Items: classifiedRefs.others, Limit: genericRefLimit},
		{Kind: "tests", Title: "Related Tests", Items: classifiedRefs.tests, Limit: genericTestLimit, IsTest: true},
	})
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

// classifyJSFamilySymbolRefs は TS/JS 共通の参照分類 owner。
// structured impact と通常 symbol search が同じテスト分離・参照分類を使うための入口。
func classifyJSFamilySymbolRefs(refs []genericSymbolRef, symbol string) jsFamilySymbolRefs {
	var normalRefs, testRefs []genericSymbolRef
	for _, ref := range refs {
		if ref.IsTest {
			testRefs = append(testRefs, ref)
		} else {
			normalRefs = append(normalRefs, ref)
		}
	}

	imports, callers, typeRefs, otherRefs := classifyJSRefs(normalRefs, symbol)
	return jsFamilySymbolRefs{
		imports:  imports,
		callers:  callers,
		typeRefs: typeRefs,
		others:   otherRefs,
		tests:    testRefs,
	}
}

// classifyJSRefs は TS/JS の参照を import / caller / type ref / other に分類する。
// 判定順: import → caller → type ref → other（最初にマッチした分類に入る）。
func classifyJSRefs(refs []genericSymbolRef, symbol string) (imports, callers, typeRefs, others []genericSymbolRef) {
	escaped := regexp.QuoteMeta(symbol)
	importPat := regexp.MustCompile(`(?:^|\s)(?:import\s|from\s|require\s*\(|export\s+(?:type\s+)?\{|export\s+.*\sfrom\s)`)
	callerPat := regexp.MustCompile(`(?:\b` + escaped + `\s*(?:\?\.)?\s*\(|\bnew\s+` + escaped + `\b)`)
	typePat := regexp.MustCompile(`(?::\s*` + escaped + `\b|\bas\s+` + escaped + `\b|\bsatisfies\s+` + escaped + `\b|extends\s+` + escaped + `\b|implements\s+` + escaped + `\b|<\s*` + escaped + `\b|\b` + escaped + `\s*\[\])`)
	genericTypeArgPat := regexp.MustCompile(`<[^>\n]*,\s*` + escaped + `\b[^>\n]*>`)

	for _, ref := range refs {
		s := ref.Snippet
		switch {
		case importPat.MatchString(s):
			imports = append(imports, ref)
		case callerPat.MatchString(s):
			callers = append(callers, ref)
		case typePat.MatchString(s) || genericTypeArgPat.MatchString(s):
			typeRefs = append(typeRefs, ref)
		default:
			others = append(others, ref)
		}
	}
	return
}

// formatJSSymbolResult は TS/JS の分類済みシンボル結果をフォーマットする。
// 出力形式は Go fast path と同様のセクション構造で、locator ID を付与する。
func formatJSSymbolResult(bundle *SymbolBundle, reg *locator.Registry) string {
	return formatSymbolBundle(bundle, reg, nil)
}
