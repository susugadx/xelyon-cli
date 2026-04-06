package search

import (
	"fmt"
	"regexp"

	"github.com/susugadx/xelyon-cli/internal/locator"
)

const (
	pyImportLimit    = 5
	pyCallerLimit    = 10
	pyDecoratorLimit = 5
)

// resolvePythonSymbol は Python 向けの enhanced symbol fast path。
// 参照を import / caller / decorator / other に分類する。
func resolvePythonSymbol(symbol string, opts SearchOptions) genericResolveResult {
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

	var normalRefs, testRefs []genericSymbolRef
	for _, ref := range filteredRefs {
		if ref.IsTest {
			testRefs = append(testRefs, ref)
		} else {
			normalRefs = append(normalRefs, ref)
		}
	}

	imports, callers, decorators, otherRefs := classifyPythonRefs(normalRefs, symbol)
	bundle := buildGenericSymbolBundle("python", symbol, def, []string{
		fmt.Sprintf("%d: %s", def.Line, def.Signature),
	}, []symbolBundleSectionInput{
		{Kind: "imports", Title: "Imports", Items: imports, Limit: pyImportLimit},
		{Kind: "callers", Title: "Callers", Items: callers, Limit: pyCallerLimit},
		{Kind: "decorators", Title: "Decorators", Items: decorators, Limit: pyDecoratorLimit},
		{Kind: "references", Title: "References", Items: otherRefs, Limit: genericRefLimit},
		{Kind: "tests", Title: "Related Tests", Items: testRefs, Limit: genericTestLimit, IsTest: true},
	})
	bundle.Debug.FileRootPath = invocationCWDOrGetwd(opts)
	return genericResolveResult{Output: formatPythonSymbolResult(bundle, opts.LocatorRegistry), Status: genericSymbolSingle, Bundle: bundle}
}

// classifyPythonRefs は Python の参照を import / caller / decorator / other に分類する。
// 判定順: import → decorator → caller → other。
func classifyPythonRefs(refs []genericSymbolRef, symbol string) (imports, callers, decorators, others []genericSymbolRef) {
	escaped := regexp.QuoteMeta(symbol)
	importPat := regexp.MustCompile(`(?:^|\s)(?:import\s+` + escaped + `\b|from\s+.*\s+import\s+.*\b` + escaped + `\b|from\s+` + escaped + `[\s.])`)
	decoratorPat := regexp.MustCompile(`^\s*@` + escaped + `\b`)
	callerPat := regexp.MustCompile(`\b` + escaped + `\s*\(`)

	for _, ref := range refs {
		s := ref.Snippet
		switch {
		case importPat.MatchString(s):
			imports = append(imports, ref)
		case decoratorPat.MatchString(s):
			decorators = append(decorators, ref)
		case callerPat.MatchString(s):
			callers = append(callers, ref)
		default:
			others = append(others, ref)
		}
	}
	return
}

// formatPythonSymbolResult は Python の分類済みシンボル結果をフォーマットする。
func formatPythonSymbolResult(bundle *SymbolBundle, reg *locator.Registry) string {
	return formatSymbolBundle(bundle, reg, nil)
}
