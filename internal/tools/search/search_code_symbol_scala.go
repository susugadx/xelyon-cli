package search

import (
	"regexp"
)

const (
	scalaImportLimit      = 5
	scalaCallerLimit      = 10
	scalaInheritanceLimit = 5
)

// resolveScalaSymbol は Scala 向けの enhanced symbol fast path。
func resolveScalaSymbol(symbol string, opts SearchOptions) genericResolveResult {
	return resolveGenericEnhancedSymbol(symbol, opts, genericEnhancedSymbolSpec{
		language:      "scala",
		buildSections: buildScalaSymbolSections,
	})
}

func buildScalaSymbolSections(normalRefs []genericSymbolRef, testRefs []genericSymbolRef, symbol string) []symbolBundleSectionInput {
	imports, callers, inheritance, otherRefs := classifyScalaRefs(normalRefs, symbol)
	return []symbolBundleSectionInput{
		{Kind: "imports", Title: "Imports", Items: imports, Limit: scalaImportLimit},
		{Kind: "callers", Title: "Callers", Items: callers, Limit: scalaCallerLimit},
		{Kind: "inheritance", Title: "Inheritance", Items: inheritance, Limit: scalaInheritanceLimit},
		{Kind: "references", Title: "References", Items: otherRefs, Limit: genericRefLimit},
		{Kind: "tests", Title: "Related Tests", Items: testRefs, Limit: genericTestLimit, IsTest: true},
	}
}

// classifyScalaRefs は Scala の参照を分類する。
func classifyScalaRefs(refs []genericSymbolRef, symbol string) (imports, callers, inheritance, others []genericSymbolRef) {
	escaped := regexp.QuoteMeta(symbol)
	importPat := regexp.MustCompile(`^\s*import\s+`)
	inheritPat := regexp.MustCompile(`(?:extends|with)\s+.*\b` + escaped + `\b`)
	callerPat := regexp.MustCompile(`\b` + escaped + `\s*[\(.]|\bnew\s+` + escaped + `\b`)

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
