package search

import (
	"regexp"
)

const (
	luaRequireLimit = 5
	luaCallerLimit  = 10
)

// resolveLuaSymbol は Lua 向けの enhanced symbol fast path。
func resolveLuaSymbol(symbol string, opts SearchOptions) genericResolveResult {
	return resolveGenericEnhancedSymbol(symbol, opts, genericEnhancedSymbolSpec{
		language:      "lua",
		buildSections: buildLuaSymbolSections,
	})
}

func buildLuaSymbolSections(normalRefs []genericSymbolRef, testRefs []genericSymbolRef, symbol string) []symbolBundleSectionInput {
	requires, callers, otherRefs := classifyLuaRefs(normalRefs, symbol)
	return []symbolBundleSectionInput{
		{Kind: "requires", Title: "Requires", Items: requires, Limit: luaRequireLimit},
		{Kind: "callers", Title: "Callers", Items: callers, Limit: luaCallerLimit},
		{Kind: "references", Title: "References", Items: otherRefs, Limit: genericRefLimit},
		{Kind: "tests", Title: "Related Tests", Items: testRefs, Limit: genericTestLimit, IsTest: true},
	}
}

// classifyLuaRefs は Lua の参照を分類する。
func classifyLuaRefs(refs []genericSymbolRef, symbol string) (requires, callers, others []genericSymbolRef) {
	escaped := regexp.QuoteMeta(symbol)
	requirePat := regexp.MustCompile(`\brequire\s*[\("]`)
	callerPat := regexp.MustCompile(`\b` + escaped + `\s*[\(.:]\b?`)

	for _, ref := range refs {
		s := ref.Snippet
		switch {
		case requirePat.MatchString(s):
			requires = append(requires, ref)
		case callerPat.MatchString(s):
			callers = append(callers, ref)
		default:
			others = append(others, ref)
		}
	}
	return
}
