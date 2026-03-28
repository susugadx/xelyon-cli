package search

import (
	"fmt"
	"regexp"

	"github.com/susugadx/xelyon-cli/internal/locator"
)

const (
	swiftImportLimit      = 5
	swiftCallerLimit      = 10
	swiftInheritanceLimit = 5
)

// resolveSwiftSymbol は Swift 向けの enhanced symbol fast path。
func resolveSwiftSymbol(symbol string, opts SearchOptions) genericResolveResult {
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

	imports, callers, inheritance, otherRefs := classifySwiftRefs(normalRefs, symbol)
	bundle := buildGenericSymbolBundle("swift", symbol, def, []string{
		fmt.Sprintf("%d: %s", def.Line, def.Signature),
	}, []symbolBundleSectionInput{
		{Kind: "imports", Title: "Imports", Items: imports, Limit: swiftImportLimit},
		{Kind: "callers", Title: "Callers", Items: callers, Limit: swiftCallerLimit},
		{Kind: "inheritance", Title: "Protocol/Inheritance", Items: inheritance, Limit: swiftInheritanceLimit},
		{Kind: "references", Title: "References", Items: otherRefs, Limit: genericRefLimit},
		{Kind: "tests", Title: "Related Tests", Items: testRefs, Limit: genericTestLimit, IsTest: true},
	})
	return genericResolveResult{Output: formatSwiftSymbolResult(bundle, opts.LocatorRegistry), Status: genericSymbolSingle, Bundle: bundle}
}

// classifySwiftRefs は Swift の参照を分類する。
func classifySwiftRefs(refs []genericSymbolRef, symbol string) (imports, callers, inheritance, others []genericSymbolRef) {
	escaped := regexp.QuoteMeta(symbol)
	importPat := regexp.MustCompile(`^\s*import\s+`)
	inheritPat := regexp.MustCompile(`:\s*.*\b` + escaped + `\b|\bextension\s+` + escaped + `\b`)
	callerPat := regexp.MustCompile(`\b` + escaped + `\s*[\(.]`)

	for _, ref := range refs {
		s := ref.Snippet
		switch {
		case importPat.MatchString(s):
			imports = append(imports, ref)
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

// formatSwiftSymbolResult は Swift の分類済みシンボル結果をフォーマットする。
func formatSwiftSymbolResult(bundle *SymbolBundle, reg *locator.Registry) string {
	return formatSymbolBundle(bundle, reg, nil)
}
