package search

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/locator"
)

const (
	swiftImportLimit      = 5
	swiftCallerLimit      = 10
	swiftInheritanceLimit = 5
)

// resolveSwiftSymbol は Swift 向けの enhanced symbol fast path。
func resolveSwiftSymbol(symbol string, opts SearchOptions) (string, genericSymbolStatus) {
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

	imports, callers, inheritance, otherRefs := classifySwiftRefs(normalRefs, symbol)
	return formatSwiftSymbolResult(def, imports, callers, inheritance, otherRefs, testRefs, opts.LocatorRegistry), genericSymbolSingle
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
func formatSwiftSymbolResult(def genericSymbolDef, imports, callers, inheritance, otherRefs, tests []genericSymbolRef, reg *locator.Registry) string {
	var sb strings.Builder

	header := fmt.Sprintf("── %s %s (L%d) in %s", def.Kind, def.Name, def.Line, def.File)
	if reg != nil {
		id := reg.Register(locator.Location{FilePath: def.File, Line: def.Line, Name: fmt.Sprintf("%s %s", def.Kind, def.Name)})
		header += " " + id
	}
	fmt.Fprintf(&sb, "%s ──\n", header)
	fmt.Fprintf(&sb, "%d: %s\n", def.Line, def.Signature)

	writeRefSection(&sb, "Imports", imports, swiftImportLimit, reg)
	writeRefSection(&sb, "Callers", callers, swiftCallerLimit, reg)
	writeRefSection(&sb, "Protocol/Inheritance", inheritance, swiftInheritanceLimit, reg)
	writeRefSection(&sb, "References", otherRefs, genericRefLimit, reg)
	writeRefSection(&sb, "Related Tests", tests, genericTestLimit, reg)

	total := len(imports) + len(callers) + len(inheritance) + len(otherRefs) + len(tests)
	if total == 0 {
		sb.WriteString("\nNo references found.\n")
	}
	return sb.String()
}
