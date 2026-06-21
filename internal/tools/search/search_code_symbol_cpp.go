package search

import (
	"regexp"
)

const (
	cppIncludeLimit     = 5
	cppCallerLimit      = 10
	cppInheritanceLimit = 5
)

// resolveCppSymbol は C/C++ 向けの enhanced symbol fast path。
func resolveCppSymbol(symbol string, opts SearchOptions) genericResolveResult {
	return resolveGenericEnhancedSymbol(symbol, opts, genericEnhancedSymbolSpec{
		language:      "cpp",
		buildSections: buildCppSymbolSections,
	})
}

func buildCppSymbolSections(normalRefs []genericSymbolRef, testRefs []genericSymbolRef, symbol string) []symbolBundleSectionInput {
	includes, callers, inheritance, otherRefs := classifyCppRefs(normalRefs, symbol)
	return []symbolBundleSectionInput{
		{Kind: "includes", Title: "Includes", Items: includes, Limit: cppIncludeLimit},
		{Kind: "callers", Title: "Callers", Items: callers, Limit: cppCallerLimit},
		{Kind: "inheritance", Title: "Inheritance", Items: inheritance, Limit: cppInheritanceLimit},
		{Kind: "references", Title: "References", Items: otherRefs, Limit: genericRefLimit},
		{Kind: "tests", Title: "Related Tests", Items: testRefs, Limit: genericTestLimit, IsTest: true},
	}
}

// classifyCppRefs は C/C++ の参照を分類する。
func classifyCppRefs(refs []genericSymbolRef, symbol string) (includes, callers, inheritance, others []genericSymbolRef) {
	escaped := regexp.QuoteMeta(symbol)
	includePat := regexp.MustCompile(`^\s*#\s*include\s+`)
	inheritPat := regexp.MustCompile(`:\s*(?:public|private|protected)\s+` + escaped + `\b|\bvirtual\s+` + escaped + `\b`)
	callerPat := regexp.MustCompile(`\b` + escaped + `\s*[\(.]|\b` + escaped + `::|\bnew\s+` + escaped + `\b`)

	for _, ref := range refs {
		s := ref.Snippet
		switch {
		case includePat.MatchString(s):
			includes = append(includes, ref)
		case inheritPat.MatchString(s):
			inheritance = append(inheritance, ref)
		case callerPat.MatchString(s):
			callers = append(callers, ref)
		default:
			others = append(others, ref)
		}
	}
	return
}
