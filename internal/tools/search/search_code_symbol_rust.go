package search

import (
	"regexp"
)

const (
	rsUseLimit    = 5
	rsCallerLimit = 10
	rsImplLimit   = 5
)

// resolveRustSymbol は Rust 向けの enhanced symbol fast path。
// 参照を use / caller / impl / other に分類する。
func resolveRustSymbol(symbol string, opts SearchOptions) genericResolveResult {
	return resolveGenericEnhancedSymbol(symbol, opts, genericEnhancedSymbolSpec{
		language:      "rust",
		buildSections: buildRustSymbolSections,
	})
}

func buildRustSymbolSections(normalRefs []genericSymbolRef, testRefs []genericSymbolRef, symbol string) []symbolBundleSectionInput {
	uses, callers, implRefs, otherRefs := classifyRustRefs(normalRefs, symbol)
	return []symbolBundleSectionInput{
		{Kind: "uses", Title: "Uses", Items: uses, Limit: rsUseLimit},
		{Kind: "callers", Title: "Callers", Items: callers, Limit: rsCallerLimit},
		{Kind: "impl_refs", Title: "Impl/Trait", Items: implRefs, Limit: rsImplLimit},
		{Kind: "references", Title: "References", Items: otherRefs, Limit: genericRefLimit},
		{Kind: "tests", Title: "Related Tests", Items: testRefs, Limit: genericTestLimit, IsTest: true},
	}
}

// classifyRustRefs は Rust の参照を use / caller / impl / other に分類する。
// 判定順: use → impl → caller → other。
func classifyRustRefs(refs []genericSymbolRef, symbol string) (uses, callers, implRefs, others []genericSymbolRef) {
	escaped := regexp.QuoteMeta(symbol)
	usePat := regexp.MustCompile(`\buse\s+.*\b` + escaped + `\b`)
	implPat := regexp.MustCompile(`(?:\bimpl\s+(?:.*\s+for\s+)?` + escaped + `\b|\bimpl\s+` + escaped + `\b|\bdyn\s+` + escaped + `\b)`)
	callerPat := regexp.MustCompile(`\b` + escaped + `\s*(?:\(|!\(|::\w)`)

	for _, ref := range refs {
		s := ref.Snippet
		switch {
		case usePat.MatchString(s):
			uses = append(uses, ref)
		case implPat.MatchString(s):
			implRefs = append(implRefs, ref)
		case callerPat.MatchString(s):
			callers = append(callers, ref)
		default:
			others = append(others, ref)
		}
	}
	return
}
