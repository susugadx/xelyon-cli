package search

import (
	"regexp"
)

const (
	pyImportLimit    = 5
	pyCallerLimit    = 10
	pyDecoratorLimit = 5
)

// resolvePythonSymbol は Python 向けの enhanced symbol fast path。
// 参照を import / caller / decorator / other に分類する。
func resolvePythonSymbol(symbol string, opts SearchOptions) genericResolveResult {
	return resolveGenericEnhancedSymbol(symbol, opts, genericEnhancedSymbolSpec{
		language:      "python",
		buildSections: buildPythonSymbolSections,
	})
}

func buildPythonSymbolSections(normalRefs []genericSymbolRef, testRefs []genericSymbolRef, symbol string) []symbolBundleSectionInput {
	imports, callers, decorators, otherRefs := classifyPythonRefs(normalRefs, symbol)
	return []symbolBundleSectionInput{
		{Kind: "imports", Title: "Imports", Items: imports, Limit: pyImportLimit},
		{Kind: "callers", Title: "Callers", Items: callers, Limit: pyCallerLimit},
		{Kind: "decorators", Title: "Decorators", Items: decorators, Limit: pyDecoratorLimit},
		{Kind: "references", Title: "References", Items: otherRefs, Limit: genericRefLimit},
		{Kind: "tests", Title: "Related Tests", Items: testRefs, Limit: genericTestLimit, IsTest: true},
	}
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
