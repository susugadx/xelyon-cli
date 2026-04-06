package search

import (
	"fmt"
	"regexp"

	"github.com/susugadx/xelyon-cli/internal/locator"
)

const (
	rsUseLimit    = 5
	rsCallerLimit = 10
	rsImplLimit   = 5
)

// resolveRustSymbol は Rust 向けの enhanced symbol fast path。
// 参照を use / caller / impl / other に分類する。
func resolveRustSymbol(symbol string, opts SearchOptions) genericResolveResult {
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

	uses, callers, implRefs, otherRefs := classifyRustRefs(normalRefs, symbol)
	bundle := buildGenericSymbolBundle("rust", symbol, def, []string{
		fmt.Sprintf("%d: %s", def.Line, def.Signature),
	}, []symbolBundleSectionInput{
		{Kind: "uses", Title: "Uses", Items: uses, Limit: rsUseLimit},
		{Kind: "callers", Title: "Callers", Items: callers, Limit: rsCallerLimit},
		{Kind: "impl_refs", Title: "Impl/Trait", Items: implRefs, Limit: rsImplLimit},
		{Kind: "references", Title: "References", Items: otherRefs, Limit: genericRefLimit},
		{Kind: "tests", Title: "Related Tests", Items: testRefs, Limit: genericTestLimit, IsTest: true},
	})
	bundle.Debug.FileRootPath = invocationCWDOrGetwd(opts)
	return genericResolveResult{Output: formatRustSymbolResult(bundle, opts.LocatorRegistry), Status: genericSymbolSingle, Bundle: bundle}
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

// formatRustSymbolResult は Rust の分類済みシンボル結果をフォーマットする。
func formatRustSymbolResult(bundle *SymbolBundle, reg *locator.Registry) string {
	return formatSymbolBundle(bundle, reg, nil)
}
