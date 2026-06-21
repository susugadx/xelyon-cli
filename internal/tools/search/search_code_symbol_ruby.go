package search

import (
	"regexp"
)

const (
	rbRequireLimit = 5
	rbCallerLimit  = 10
	rbMixinLimit   = 5
)

// resolveRubySymbol は Ruby 向けの enhanced symbol fast path。
func resolveRubySymbol(symbol string, opts SearchOptions) genericResolveResult {
	return resolveGenericEnhancedSymbol(symbol, opts, genericEnhancedSymbolSpec{
		language:      "ruby",
		buildSections: buildRubySymbolSections,
	})
}

func buildRubySymbolSections(normalRefs []genericSymbolRef, testRefs []genericSymbolRef, symbol string) []symbolBundleSectionInput {
	requires, callers, mixins, otherRefs := classifyRubyRefs(normalRefs, symbol)
	return []symbolBundleSectionInput{
		{Kind: "requires", Title: "Requires", Items: requires, Limit: rbRequireLimit},
		{Kind: "callers", Title: "Callers", Items: callers, Limit: rbCallerLimit},
		{Kind: "mixins", Title: "Mixins/Inheritance", Items: mixins, Limit: rbMixinLimit},
		{Kind: "references", Title: "References", Items: otherRefs, Limit: genericRefLimit},
		{Kind: "tests", Title: "Related Tests", Items: testRefs, Limit: genericTestLimit, IsTest: true},
	}
}

// classifyRubyRefs は Ruby の参照を分類する。
func classifyRubyRefs(refs []genericSymbolRef, symbol string) (requires, callers, mixins, others []genericSymbolRef) {
	escaped := regexp.QuoteMeta(symbol)
	requirePat := regexp.MustCompile(`(?:require|require_relative|load)\s+`)
	mixinPat := regexp.MustCompile(`(?:include|extend|prepend)\s+` + escaped + `\b|<\s*` + escaped + `\b`)
	callerPat := regexp.MustCompile(`\b` + escaped + `\s*[\(.]|\b` + escaped + `\.new\b`)

	for _, ref := range refs {
		s := ref.Snippet
		switch {
		case requirePat.MatchString(s):
			requires = append(requires, ref)
		case mixinPat.MatchString(s):
			mixins = append(mixins, ref)
		case callerPat.MatchString(s):
			callers = append(callers, ref)
		default:
			others = append(others, ref)
		}
	}
	return
}
