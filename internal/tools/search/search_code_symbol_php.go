package search

import (
	"fmt"
	"regexp"

	"github.com/susugadx/xelyon-cli/internal/locator"
)

const (
	phpUseLimit         = 5
	phpCallerLimit      = 10
	phpInheritanceLimit = 5
)

// resolvePHPSymbol は PHP 向けの enhanced symbol fast path。
// 参照を use / caller / inheritance / other に分類する。
func resolvePHPSymbol(symbol string, opts SearchOptions) genericResolveResult {
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

	uses, callers, inheritance, otherRefs := classifyPHPRefs(normalRefs, symbol)
	bundle := buildGenericSymbolBundle("php", symbol, def, []string{
		fmt.Sprintf("%d: %s", def.Line, def.Signature),
	}, []symbolBundleSectionInput{
		{Kind: "uses", Title: "Uses", Items: uses, Limit: phpUseLimit},
		{Kind: "callers", Title: "Callers", Items: callers, Limit: phpCallerLimit},
		{Kind: "inheritance", Title: "Inheritance", Items: inheritance, Limit: phpInheritanceLimit},
		{Kind: "references", Title: "References", Items: otherRefs, Limit: genericRefLimit},
		{Kind: "tests", Title: "Related Tests", Items: testRefs, Limit: genericTestLimit, IsTest: true},
	})
	return genericResolveResult{Output: formatPHPSymbolResult(bundle, opts.LocatorRegistry), Status: genericSymbolSingle, Bundle: bundle}
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

// formatPHPSymbolResult は PHP の分類済みシンボル結果をフォーマットする。
func formatPHPSymbolResult(bundle *SymbolBundle, reg *locator.Registry) string {
	return formatSymbolBundle(bundle, reg, nil)
}
