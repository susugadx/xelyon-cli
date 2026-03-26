package search

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/locator"
)

const (
	csUsingLimit       = 5
	csCallerLimit      = 10
	csAttributeLimit   = 5
	csInheritanceLimit = 5
)

// resolveCSharpSymbol は C# 向けの enhanced symbol fast path。
// 参照を using / caller / attribute / inheritance / other に分類する。
func resolveCSharpSymbol(symbol string, opts SearchOptions) (string, genericSymbolStatus) {
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

	usings, callers, attributes, inheritance, otherRefs := classifyCSharpRefs(normalRefs, symbol)
	return formatCSharpSymbolResult(def, usings, callers, attributes, inheritance, otherRefs, testRefs, opts.LocatorRegistry), genericSymbolSingle
}

// classifyCSharpRefs は C# の参照を分類する。
// 判定順: using → attribute → inheritance → caller → other。
func classifyCSharpRefs(refs []genericSymbolRef, symbol string) (usings, callers, attributes, inheritance, others []genericSymbolRef) {
	escaped := regexp.QuoteMeta(symbol)
	usingPat := regexp.MustCompile(`^\s*using\s+`)
	attrPat := regexp.MustCompile(`\[\s*` + escaped + `\b`)
	inheritPat := regexp.MustCompile(`:\s*.*\b` + escaped + `\b|where\s+\w+\s*:\s*` + escaped + `\b`)
	callerPat := regexp.MustCompile(`\b` + escaped + `\s*[\(.]|\bnew\s+` + escaped + `\b`)

	for _, ref := range refs {
		s := ref.Snippet
		switch {
		case usingPat.MatchString(s):
			usings = append(usings, ref)
		case attrPat.MatchString(s):
			attributes = append(attributes, ref)
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

// formatCSharpSymbolResult は C# の分類済みシンボル結果をフォーマットする。
func formatCSharpSymbolResult(def genericSymbolDef, usings, callers, attributes, inheritance, otherRefs, tests []genericSymbolRef, reg *locator.Registry) string {
	var sb strings.Builder

	header := fmt.Sprintf("── %s %s (L%d) in %s", def.Kind, def.Name, def.Line, def.File)
	if reg != nil {
		id := reg.Register(locator.Location{
			FilePath: def.File,
			Line:     def.Line,
			Name:     fmt.Sprintf("%s %s", def.Kind, def.Name),
		})
		header += " " + id
	}
	fmt.Fprintf(&sb, "%s ──\n", header)
	fmt.Fprintf(&sb, "%d: %s\n", def.Line, def.Signature)

	writeRefSection(&sb, "Usings", usings, csUsingLimit, reg)
	writeRefSection(&sb, "Callers", callers, csCallerLimit, reg)
	writeRefSection(&sb, "Attributes", attributes, csAttributeLimit, reg)
	writeRefSection(&sb, "Inheritance", inheritance, csInheritanceLimit, reg)
	writeRefSection(&sb, "References", otherRefs, genericRefLimit, reg)
	writeRefSection(&sb, "Related Tests", tests, genericTestLimit, reg)

	total := len(usings) + len(callers) + len(attributes) + len(inheritance) + len(otherRefs) + len(tests)
	if total == 0 {
		sb.WriteString("\nNo references found.\n")
	}

	return sb.String()
}
