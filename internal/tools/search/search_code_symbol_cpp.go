package search

import (
	"fmt"
	"regexp"

	"github.com/susugadx/xelyon-cli/internal/locator"
)

const (
	cppIncludeLimit     = 5
	cppCallerLimit      = 10
	cppInheritanceLimit = 5
)

// resolveCppSymbol は C/C++ 向けの enhanced symbol fast path。
func resolveCppSymbol(symbol string, opts SearchOptions) genericResolveResult {
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

	includes, callers, inheritance, otherRefs := classifyCppRefs(normalRefs, symbol)
	bundle := buildGenericSymbolBundle("cpp", symbol, def, []string{
		fmt.Sprintf("%d: %s", def.Line, def.Signature),
	}, []symbolBundleSectionInput{
		{Kind: "includes", Title: "Includes", Items: includes, Limit: cppIncludeLimit},
		{Kind: "callers", Title: "Callers", Items: callers, Limit: cppCallerLimit},
		{Kind: "inheritance", Title: "Inheritance", Items: inheritance, Limit: cppInheritanceLimit},
		{Kind: "references", Title: "References", Items: otherRefs, Limit: genericRefLimit},
		{Kind: "tests", Title: "Related Tests", Items: testRefs, Limit: genericTestLimit, IsTest: true},
	})
	bundle.Debug.FileRootPath = invocationCWDOrGetwd(opts)
	return genericResolveResult{Output: formatCppSymbolResult(bundle, opts.LocatorRegistry), Status: genericSymbolSingle, Bundle: bundle}
}

// classifyCppRefs は C/C++ の参照を分類する。
func classifyCppRefs(refs []genericSymbolRef, symbol string) (includes, callers, inheritance, others []genericSymbolRef) {
	escaped := regexp.QuoteMeta(symbol)
	includePat := regexp.MustCompile(`^\s*#\s*include\s+`)
	inheritPat := regexp.MustCompile(`:\s*(?:public|private|protected)\s+` + escaped + `\b|\bvirtual\s+` + escaped + `\b`)
	callerPat := regexp.MustCompile(`\b` + escaped + `\s*[\(.]|\b` + escaped + `::|\bnew\s+` + escaped + `\b`)

	for _, ref := range refs {
		s := ref.Snippet
		switch {
		case includePat.MatchString(s):
			includes = append(includes, ref)
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

// formatCppSymbolResult は C/C++ の分類済みシンボル結果をフォーマットする。
func formatCppSymbolResult(bundle *SymbolBundle, reg *locator.Registry) string {
	return formatSymbolBundle(bundle, reg, nil)
}
