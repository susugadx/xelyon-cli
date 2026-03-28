package search

import (
	"fmt"
	"regexp"

	"github.com/susugadx/xelyon-cli/internal/locator"
)

const (
	rbRequireLimit = 5
	rbCallerLimit  = 10
	rbMixinLimit   = 5
)

// resolveRubySymbol は Ruby 向けの enhanced symbol fast path。
func resolveRubySymbol(symbol string, opts SearchOptions) genericResolveResult {
	defs := findGenericDefinitions(symbol, opts)
	if len(defs) == 0 {
		return genericResolveResult{Status: genericSymbolNone}
	}
	if len(defs) > 1 {
		return genericResolveResult{Output: formatGenericMultipleDefs(symbol, defs, opts.LocatorRegistry), Status: genericSymbolMultiple}
	}

	def := defs[0]
	refs := findGenericReferences(symbol, opts)
	filteredRefs := filterGenericRefs(refs, def)

	var normalRefs, testRefs []genericSymbolRef
	for _, ref := range filteredRefs {
		if ref.IsTest {
			testRefs = append(testRefs, ref)
		} else {
			normalRefs = append(normalRefs, ref)
		}
	}

	requires, callers, mixins, otherRefs := classifyRubyRefs(normalRefs, symbol)
	bundle := buildGenericSymbolBundle("ruby", symbol, def, []string{
		fmt.Sprintf("%d: %s", def.Line, def.Signature),
	}, []symbolBundleSectionInput{
		{Kind: "requires", Title: "Requires", Items: requires, Limit: rbRequireLimit},
		{Kind: "callers", Title: "Callers", Items: callers, Limit: rbCallerLimit},
		{Kind: "mixins", Title: "Mixins/Inheritance", Items: mixins, Limit: rbMixinLimit},
		{Kind: "references", Title: "References", Items: otherRefs, Limit: genericRefLimit},
		{Kind: "tests", Title: "Related Tests", Items: testRefs, Limit: genericTestLimit, IsTest: true},
	})
	return genericResolveResult{Output: formatRubySymbolResult(bundle, opts.LocatorRegistry), Status: genericSymbolSingle, Bundle: bundle}
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

// formatRubySymbolResult は Ruby の分類済みシンボル結果をフォーマットする。
func formatRubySymbolResult(bundle *SymbolBundle, reg *locator.Registry) string {
	return formatSymbolBundle(bundle, reg, nil)
}
