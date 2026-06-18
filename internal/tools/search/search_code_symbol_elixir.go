package search

import (
	"regexp"
)

const (
	exAliasLimit     = 5
	exCallerLimit    = 10
	exBehaviourLimit = 5
)

// resolveElixirSymbol は Elixir 向けの enhanced symbol fast path。
func resolveElixirSymbol(symbol string, opts SearchOptions) genericResolveResult {
	return resolveGenericEnhancedSymbol(symbol, opts, genericEnhancedSymbolSpec{
		language:      "elixir",
		buildSections: buildElixirSymbolSections,
	})
}

func buildElixirSymbolSections(normalRefs []genericSymbolRef, testRefs []genericSymbolRef, symbol string) []symbolBundleSectionInput {
	aliases, callers, behaviours, otherRefs := classifyElixirRefs(normalRefs, symbol)
	return []symbolBundleSectionInput{
		{Kind: "aliases", Title: "Aliases/Imports", Items: aliases, Limit: exAliasLimit},
		{Kind: "callers", Title: "Callers", Items: callers, Limit: exCallerLimit},
		{Kind: "behaviours", Title: "Behaviours", Items: behaviours, Limit: exBehaviourLimit},
		{Kind: "references", Title: "References", Items: otherRefs, Limit: genericRefLimit},
		{Kind: "tests", Title: "Related Tests", Items: testRefs, Limit: genericTestLimit, IsTest: true},
	}
}

// classifyElixirRefs は Elixir の参照を分類する。
func classifyElixirRefs(refs []genericSymbolRef, symbol string) (aliases, callers, behaviours, others []genericSymbolRef) {
	escaped := regexp.QuoteMeta(symbol)
	aliasPat := regexp.MustCompile(`^\s*(?:alias|import|use|require)\s+.*\b` + escaped + `\b`)
	behaviourPat := regexp.MustCompile(`@(?:behaviour|impl)\s+` + escaped + `\b`)
	callerPat := regexp.MustCompile(`\b` + escaped + `\.`)

	for _, ref := range refs {
		s := ref.Snippet
		switch {
		case aliasPat.MatchString(s):
			aliases = append(aliases, ref)
		case behaviourPat.MatchString(s):
			behaviours = append(behaviours, ref)
		case callerPat.MatchString(s):
			callers = append(callers, ref)
		default:
			others = append(others, ref)
		}
	}
	return
}
