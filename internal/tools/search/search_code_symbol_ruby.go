package search

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/locator"
)

const (
	rbRequireLimit = 5
	rbCallerLimit  = 10
	rbMixinLimit   = 5
)

// resolveRubySymbol は Ruby 向けの enhanced symbol fast path。
func resolveRubySymbol(symbol string, opts SearchOptions) (string, genericSymbolStatus) {
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

	requires, callers, mixins, otherRefs := classifyRubyRefs(normalRefs, symbol)
	return formatRubySymbolResult(def, requires, callers, mixins, otherRefs, testRefs, opts.LocatorRegistry), genericSymbolSingle
}

// classifyRubyRefs は Ruby の参照を分類する。
func classifyRubyRefs(refs []genericSymbolRef, symbol string) (requires, callers, mixins, others []genericSymbolRef) {
	escaped := regexp.QuoteMeta(symbol)
	requirePat := regexp.MustCompile(`(?:require|require_relative|load)\s+`)
	mixinPat := regexp.MustCompile(`(?:include|extend|prepend)\s+` + escaped + `\b|<\s*` + escaped + `\b`)
	callerPat := regexp.MustCompile(`\b` + escaped + `\s*[\(.]|\b` + escaped + `\.new\b`)

	for _, ref := range refs {
		s := ref.Snippet
		switch {
		case requirePat.MatchString(s):
			requires = append(requires, ref)
		case mixinPat.MatchString(s):
			mixins = append(mixins, ref)
		case callerPat.MatchString(s):
			callers = append(callers, ref)
		default:
			others = append(others, ref)
		}
	}
	return
}

// formatRubySymbolResult は Ruby の分類済みシンボル結果をフォーマットする。
func formatRubySymbolResult(def genericSymbolDef, requires, callers, mixins, otherRefs, tests []genericSymbolRef, reg *locator.Registry) string {
	var sb strings.Builder

	header := fmt.Sprintf("── %s %s (L%d) in %s", def.Kind, def.Name, def.Line, def.File)
	if reg != nil {
		id := reg.Register(locator.Location{FilePath: def.File, Line: def.Line, Name: fmt.Sprintf("%s %s", def.Kind, def.Name)})
		header += " " + id
	}
	fmt.Fprintf(&sb, "%s ──\n", header)
	fmt.Fprintf(&sb, "%d: %s\n", def.Line, def.Signature)

	writeRefSection(&sb, "Requires", requires, rbRequireLimit, reg)
	writeRefSection(&sb, "Callers", callers, rbCallerLimit, reg)
	writeRefSection(&sb, "Mixins/Inheritance", mixins, rbMixinLimit, reg)
	writeRefSection(&sb, "References", otherRefs, genericRefLimit, reg)
	writeRefSection(&sb, "Related Tests", tests, genericTestLimit, reg)

	total := len(requires) + len(callers) + len(mixins) + len(otherRefs) + len(tests)
	if total == 0 {
		sb.WriteString("\nNo references found.\n")
	}
	return sb.String()
}
