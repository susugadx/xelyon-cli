package search

import (
	"regexp"
)

const (
	javaImportLimit      = 5
	javaCallerLimit      = 10
	javaAnnotationLimit  = 5
	javaInheritanceLimit = 5
)

// resolveJavaSymbol は Java / Kotlin 向けの enhanced symbol fast path。
// 参照を import / caller / annotation / inheritance / other に分類する。
func resolveJavaSymbol(symbol string, opts SearchOptions) genericResolveResult {
	return resolveGenericEnhancedSymbol(symbol, opts, genericEnhancedSymbolSpec{
		language:      "java",
		buildSections: buildJavaSymbolSections,
	})
}

func buildJavaSymbolSections(normalRefs []genericSymbolRef, testRefs []genericSymbolRef, symbol string) []symbolBundleSectionInput {
	imports, callers, annotations, inheritance, otherRefs := classifyJavaRefs(normalRefs, symbol)
	return []symbolBundleSectionInput{
		{Kind: "imports", Title: "Imports", Items: imports, Limit: javaImportLimit},
		{Kind: "callers", Title: "Callers", Items: callers, Limit: javaCallerLimit},
		{Kind: "annotations", Title: "Annotations", Items: annotations, Limit: javaAnnotationLimit},
		{Kind: "inheritance", Title: "Inheritance", Items: inheritance, Limit: javaInheritanceLimit},
		{Kind: "references", Title: "References", Items: otherRefs, Limit: genericRefLimit},
		{Kind: "tests", Title: "Related Tests", Items: testRefs, Limit: genericTestLimit, IsTest: true},
	}
}

// classifyJavaRefs は Java/Kotlin の参照を分類する。
// 判定順: import → annotation → inheritance → caller → other。
func classifyJavaRefs(refs []genericSymbolRef, symbol string) (imports, callers, annotations, inheritance, others []genericSymbolRef) {
	escaped := regexp.QuoteMeta(symbol)
	importPat := regexp.MustCompile(`^\s*import\s+`)
	annotationPat := regexp.MustCompile(`@` + escaped + `\b`)
	inheritPat := regexp.MustCompile(`(?:extends|implements)\s+.*\b` + escaped + `\b|:\s*.*\b` + escaped + `\b`)
	callerPat := regexp.MustCompile(`\b` + escaped + `\s*[\(.]|\bnew\s+` + escaped + `\b`)

	for _, ref := range refs {
		s := ref.Snippet
		switch {
		case importPat.MatchString(s):
			imports = append(imports, ref)
		case annotationPat.MatchString(s):
			annotations = append(annotations, ref)
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
