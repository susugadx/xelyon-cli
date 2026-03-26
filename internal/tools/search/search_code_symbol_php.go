package search

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/locator"
)

const (
	phpUseLimit         = 5
	phpCallerLimit      = 10
	phpInheritanceLimit = 5
)

// resolvePHPSymbol は PHP 向けの enhanced symbol fast path。
// 参照を use / caller / inheritance / other に分類する。
func resolvePHPSymbol(symbol string, opts SearchOptions) (string, genericSymbolStatus) {
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

	uses, callers, inheritance, otherRefs := classifyPHPRefs(normalRefs, symbol)
	return formatPHPSymbolResult(def, uses, callers, inheritance, otherRefs, testRefs, opts.LocatorRegistry), genericSymbolSingle
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
func formatPHPSymbolResult(def genericSymbolDef, uses, callers, inheritance, otherRefs, tests []genericSymbolRef, reg *locator.Registry) string {
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

	writeRefSection(&sb, "Uses", uses, phpUseLimit, reg)
	writeRefSection(&sb, "Callers", callers, phpCallerLimit, reg)
	writeRefSection(&sb, "Inheritance", inheritance, phpInheritanceLimit, reg)
	writeRefSection(&sb, "References", otherRefs, genericRefLimit, reg)
	writeRefSection(&sb, "Related Tests", tests, genericTestLimit, reg)

	total := len(uses) + len(callers) + len(inheritance) + len(otherRefs) + len(tests)
	if total == 0 {
		sb.WriteString("\nNo references found.\n")
	}

	return sb.String()
}
