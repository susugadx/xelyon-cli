package search

import (
	"fmt"
	"regexp"

	"github.com/susugadx/xelyon-cli/internal/locator"
)

const (
	exAliasLimit     = 5
	exCallerLimit    = 10
	exBehaviourLimit = 5
)

// resolveElixirSymbol は Elixir 向けの enhanced symbol fast path。
func resolveElixirSymbol(symbol string, opts SearchOptions) genericResolveResult {
	defs := findGenericDefinitions(symbol, opts)
	if len(defs) == 0 {
		return genericResolveResult{Status: genericSymbolNone}
	}
	if len(defs) > 1 {
		return genericResolveResult{Output: formatGenericMultipleDefsWithOptions(symbol, defs, opts.LocatorRegistry, opts), Status: genericSymbolMultiple}
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

	aliases, callers, behaviours, otherRefs := classifyElixirRefs(normalRefs, symbol)
	bundle := buildGenericSymbolBundle("elixir", symbol, def, []string{
		fmt.Sprintf("%d: %s", def.Line, def.Signature),
	}, []symbolBundleSectionInput{
		{Kind: "aliases", Title: "Aliases/Imports", Items: aliases, Limit: exAliasLimit},
		{Kind: "callers", Title: "Callers", Items: callers, Limit: exCallerLimit},
		{Kind: "behaviours", Title: "Behaviours", Items: behaviours, Limit: exBehaviourLimit},
		{Kind: "references", Title: "References", Items: otherRefs, Limit: genericRefLimit},
		{Kind: "tests", Title: "Related Tests", Items: testRefs, Limit: genericTestLimit, IsTest: true},
	})
	bundle.Debug.FileRootPath = invocationCWDOrGetwd(opts)
	return genericResolveResult{Output: formatElixirSymbolResult(bundle, opts.LocatorRegistry), Status: genericSymbolSingle, Bundle: bundle}
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

// formatElixirSymbolResult は Elixir の分類済みシンボル結果をフォーマットする。
func formatElixirSymbolResult(bundle *SymbolBundle, reg *locator.Registry) string {
	return formatSymbolBundle(bundle, reg, nil)
}
