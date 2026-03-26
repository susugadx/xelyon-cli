package search

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/locator"
)

const (
	rsUseLimit    = 5
	rsCallerLimit = 10
	rsImplLimit   = 5
)

// resolveRustSymbol は Rust 向けの enhanced symbol fast path。
// 参照を use / caller / impl / other に分類する。
func resolveRustSymbol(symbol string, opts SearchOptions) (string, genericSymbolStatus) {
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

	uses, callers, implRefs, otherRefs := classifyRustRefs(normalRefs, symbol)
	return formatRustSymbolResult(def, uses, callers, implRefs, otherRefs, testRefs, opts.LocatorRegistry), genericSymbolSingle
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
func formatRustSymbolResult(def genericSymbolDef, uses, callers, implRefs, otherRefs, tests []genericSymbolRef, reg *locator.Registry) string {
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

	writeRefSection(&sb, "Uses", uses, rsUseLimit, reg)
	writeRefSection(&sb, "Callers", callers, rsCallerLimit, reg)
	writeRefSection(&sb, "Impl/Trait", implRefs, rsImplLimit, reg)
	writeRefSection(&sb, "References", otherRefs, genericRefLimit, reg)
	writeRefSection(&sb, "Related Tests", tests, genericTestLimit, reg)

	total := len(uses) + len(callers) + len(implRefs) + len(otherRefs) + len(tests)
	if total == 0 {
		sb.WriteString("\nNo references found.\n")
	}

	return sb.String()
}
