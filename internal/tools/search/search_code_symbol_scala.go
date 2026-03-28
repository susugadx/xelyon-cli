package search

import (
	"fmt"
	"regexp"

	"github.com/susugadx/xelyon-cli/internal/locator"
)

const (
	scalaImportLimit      = 5
	scalaCallerLimit      = 10
	scalaInheritanceLimit = 5
)

// resolveScalaSymbol は Scala 向けの enhanced symbol fast path。
func resolveScalaSymbol(symbol string, opts SearchOptions) genericResolveResult {
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

	imports, callers, inheritance, otherRefs := classifyScalaRefs(normalRefs, symbol)
	bundle := buildGenericSymbolBundle("scala", symbol, def, []string{
		fmt.Sprintf("%d: %s", def.Line, def.Signature),
	}, []symbolBundleSectionInput{
		{Kind: "imports", Title: "Imports", Items: imports, Limit: scalaImportLimit},
		{Kind: "callers", Title: "Callers", Items: callers, Limit: scalaCallerLimit},
		{Kind: "inheritance", Title: "Inheritance", Items: inheritance, Limit: scalaInheritanceLimit},
		{Kind: "references", Title: "References", Items: otherRefs, Limit: genericRefLimit},
		{Kind: "tests", Title: "Related Tests", Items: testRefs, Limit: genericTestLimit, IsTest: true},
	})
	return genericResolveResult{Output: formatScalaSymbolResult(bundle, opts.LocatorRegistry), Status: genericSymbolSingle, Bundle: bundle}
}

// classifyScalaRefs は Scala の参照を分類する。
func classifyScalaRefs(refs []genericSymbolRef, symbol string) (imports, callers, inheritance, others []genericSymbolRef) {
	escaped := regexp.QuoteMeta(symbol)
	importPat := regexp.MustCompile(`^\s*import\s+`)
	inheritPat := regexp.MustCompile(`(?:extends|with)\s+.*\b` + escaped + `\b`)
	callerPat := regexp.MustCompile(`\b` + escaped + `\s*[\(.]|\bnew\s+` + escaped + `\b`)

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

// formatScalaSymbolResult は Scala の分類済みシンボル結果をフォーマットする。
func formatScalaSymbolResult(bundle *SymbolBundle, reg *locator.Registry) string {
	return formatSymbolBundle(bundle, reg, nil)
}
