package search

import (
	"regexp"
)

const (
	csUsingLimit       = 5
	csCallerLimit      = 10
	csAttributeLimit   = 5
	csInheritanceLimit = 5
)

// resolveCSharpSymbol は C# 向けの enhanced symbol fast path。
// 参照を using / caller / attribute / inheritance / other に分類する。
func resolveCSharpSymbol(symbol string, opts SearchOptions) genericResolveResult {
	return resolveGenericEnhancedSymbol(symbol, opts, genericEnhancedSymbolSpec{
		language:      "csharp",
		buildSections: buildCSharpSymbolSections,
	})
}

func buildCSharpSymbolSections(normalRefs []genericSymbolRef, testRefs []genericSymbolRef, symbol string) []symbolBundleSectionInput {
	usings, callers, attributes, inheritance, otherRefs := classifyCSharpRefs(normalRefs, symbol)
	return []symbolBundleSectionInput{
		{Kind: "usings", Title: "Usings", Items: usings, Limit: csUsingLimit},
		{Kind: "callers", Title: "Callers", Items: callers, Limit: csCallerLimit},
		{Kind: "attributes", Title: "Attributes", Items: attributes, Limit: csAttributeLimit},
		{Kind: "inheritance", Title: "Inheritance", Items: inheritance, Limit: csInheritanceLimit},
		{Kind: "references", Title: "References", Items: otherRefs, Limit: genericRefLimit},
		{Kind: "tests", Title: "Related Tests", Items: testRefs, Limit: genericTestLimit, IsTest: true},
	}
}

// classifyCSharpRefs は C# の参照を分類する。
// 判定順: using → attribute → inheritance → caller → other。
func classifyCSharpRefs(refs []genericSymbolRef, symbol string) (usings, callers, attributes, inheritance, others []genericSymbolRef) {
	escaped := regexp.QuoteMeta(symbol)
	usingPat := regexp.MustCompile(`^\s*using\s+`)
	attrPat := regexp.MustCompile(`\[\s*` + escaped + `\b`)
	inheritPat := regexp.MustCompile(`:\s*.*\b` + escaped + `\b|where\s+\w+\s*:\s*` + escaped + `\b`)
	callerPat := regexp.MustCompile(`\b` + escaped + `\s*[\(.]|\bnew\s+` + escaped + `\b`)

	for _, ref := range refs {
		s := ref.Snippet
		switch {
		case usingPat.MatchString(s):
			usings = append(usings, ref)
		case attrPat.MatchString(s):
			attributes = append(attributes, ref)
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
