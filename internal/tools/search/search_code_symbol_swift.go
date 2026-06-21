package search

import (
	"regexp"
)

const (
	swiftImportLimit      = 5
	swiftCallerLimit      = 10
	swiftInheritanceLimit = 5
)

// resolveSwiftSymbol は Swift 向けの enhanced symbol fast path。
func resolveSwiftSymbol(symbol string, opts SearchOptions) genericResolveResult {
	return resolveGenericEnhancedSymbol(symbol, opts, genericEnhancedSymbolSpec{
		language:      "swift",
		buildSections: buildSwiftSymbolSections,
	})
}

func buildSwiftSymbolSections(normalRefs []genericSymbolRef, testRefs []genericSymbolRef, symbol string) []symbolBundleSectionInput {
	imports, callers, inheritance, otherRefs := classifySwiftRefs(normalRefs, symbol)
	return []symbolBundleSectionInput{
		{Kind: "imports", Title: "Imports", Items: imports, Limit: swiftImportLimit},
		{Kind: "callers", Title: "Callers", Items: callers, Limit: swiftCallerLimit},
		{Kind: "inheritance", Title: "Protocol/Inheritance", Items: inheritance, Limit: swiftInheritanceLimit},
		{Kind: "references", Title: "References", Items: otherRefs, Limit: genericRefLimit},
		{Kind: "tests", Title: "Related Tests", Items: testRefs, Limit: genericTestLimit, IsTest: true},
	}
}

// classifySwiftRefs は Swift の参照を分類する。
func classifySwiftRefs(refs []genericSymbolRef, symbol string) (imports, callers, inheritance, others []genericSymbolRef) {
	escaped := regexp.QuoteMeta(symbol)
	importPat := regexp.MustCompile(`^\s*import\s+`)
	inheritPat := regexp.MustCompile(`:\s*.*\b` + escaped + `\b|\bextension\s+` + escaped + `\b`)
	callerPat := regexp.MustCompile(`\b` + escaped + `\s*[\(.]`)

	for _, ref := range refs {
		s := ref.Snippet
		switch {
		case importPat.MatchString(s):
			imports = append(imports, ref)
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
