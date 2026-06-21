package search

import (
	"regexp"
)

const (
	phpUseLimit         = 5
	phpCallerLimit      = 10
	phpInheritanceLimit = 5
)

// resolvePHPSymbol は PHP 向けの enhanced symbol fast path。
// 参照を use / caller / inheritance / other に分類する。
func resolvePHPSymbol(symbol string, opts SearchOptions) genericResolveResult {
	return resolveGenericEnhancedSymbol(symbol, opts, genericEnhancedSymbolSpec{
		language:      "php",
		buildSections: buildPHPSymbolSections,
	})
}

func buildPHPSymbolSections(normalRefs []genericSymbolRef, testRefs []genericSymbolRef, symbol string) []symbolBundleSectionInput {
	uses, callers, inheritance, otherRefs := classifyPHPRefs(normalRefs, symbol)
	return []symbolBundleSectionInput{
		{Kind: "uses", Title: "Uses", Items: uses, Limit: phpUseLimit},
		{Kind: "callers", Title: "Callers", Items: callers, Limit: phpCallerLimit},
		{Kind: "inheritance", Title: "Inheritance", Items: inheritance, Limit: phpInheritanceLimit},
		{Kind: "references", Title: "References", Items: otherRefs, Limit: genericRefLimit},
		{Kind: "tests", Title: "Related Tests", Items: testRefs, Limit: genericTestLimit, IsTest: true},
	}
}

// classifyPHPRefs は PHP の参照を分類する。
// 判定順: use → inheritance → caller → other。
func classifyPHPRefs(refs []genericSymbolRef, symbol string) (uses, callers, inheritance, others []genericSymbolRef) {
	escaped := regexp.QuoteMeta(symbol)
	usePat := regexp.MustCompile(`^\s*use\s+`)
	inheritPat := regexp.MustCompile(`(?:extends|implements)\s+.*\b` + escaped + `\b`)
	callerPat := regexp.MustCompile(`\b` + escaped + `\s*[\(:]|\bnew\s+` + escaped + `\b`)

	for _, ref := range refs {
		s := ref.Snippet
		switch {
		case usePat.MatchString(s):
			uses = append(uses, ref)
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
