package search

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/locator"
)

const (
	cppIncludeLimit     = 5
	cppCallerLimit      = 10
	cppInheritanceLimit = 5
)

// resolveCppSymbol は C/C++ 向けの enhanced symbol fast path。
func resolveCppSymbol(symbol string, opts SearchOptions) (string, genericSymbolStatus) {
	defs := findGenericDefinitions(symbol, opts)
	if len(defs) == 0 {
		return "", genericSymbolNone
	}
	if len(defs) > 1 {
		return formatGenericMultipleDefs(symbol, defs, opts.LocatorRegistry), genericSymbolMultiple
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
	return formatCppSymbolResult(def, includes, callers, inheritance, otherRefs, testRefs, opts.LocatorRegistry), genericSymbolSingle
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
func formatCppSymbolResult(def genericSymbolDef, includes, callers, inheritance, otherRefs, tests []genericSymbolRef, reg *locator.Registry) string {
	var sb strings.Builder

	header := fmt.Sprintf("── %s %s (L%d) in %s", def.Kind, def.Name, def.Line, def.File)
	if reg != nil {
		id := reg.Register(locator.Location{FilePath: def.File, Line: def.Line, Name: fmt.Sprintf("%s %s", def.Kind, def.Name)})
		header += " " + id
	}
	fmt.Fprintf(&sb, "%s ──\n", header)
	fmt.Fprintf(&sb, "%d: %s\n", def.Line, def.Signature)

	writeRefSection(&sb, "Includes", includes, cppIncludeLimit, reg)
	writeRefSection(&sb, "Callers", callers, cppCallerLimit, reg)
	writeRefSection(&sb, "Inheritance", inheritance, cppInheritanceLimit, reg)
	writeRefSection(&sb, "References", otherRefs, genericRefLimit, reg)
	writeRefSection(&sb, "Related Tests", tests, genericTestLimit, reg)

	total := len(includes) + len(callers) + len(inheritance) + len(otherRefs) + len(tests)
	if total == 0 {
		sb.WriteString("\nNo references found.\n")
	}
	return sb.String()
}
