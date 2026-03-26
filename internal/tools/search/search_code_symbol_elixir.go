package search

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/locator"
)

const (
	exAliasLimit     = 5
	exCallerLimit    = 10
	exBehaviourLimit = 5
)

// resolveElixirSymbol は Elixir 向けの enhanced symbol fast path。
func resolveElixirSymbol(symbol string, opts SearchOptions) (string, genericSymbolStatus) {
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

	aliases, callers, behaviours, otherRefs := classifyElixirRefs(normalRefs, symbol)
	return formatElixirSymbolResult(def, aliases, callers, behaviours, otherRefs, testRefs, opts.LocatorRegistry), genericSymbolSingle
}

// classifyElixirRefs は Elixir の参照を分類する。
func classifyElixirRefs(refs []genericSymbolRef, symbol string) (aliases, callers, behaviours, others []genericSymbolRef) {
	escaped := regexp.QuoteMeta(symbol)
	aliasPat := regexp.MustCompile(`^\s*(?:alias|import|use|require)\s+.*\b` + escaped + `\b`)
	behaviourPat := regexp.MustCompile(`@(?:behaviour|impl)\s+` + escaped + `\b`)
	callerPat := regexp.MustCompile(`\b` + escaped + `\.`)

	for _, ref := range refs {
		s := ref.Snippet
		switch {
		case aliasPat.MatchString(s):
			aliases = append(aliases, ref)
		case behaviourPat.MatchString(s):
			behaviours = append(behaviours, ref)
		case callerPat.MatchString(s):
			callers = append(callers, ref)
		default:
			others = append(others, ref)
		}
	}
	return
}

// formatElixirSymbolResult は Elixir の分類済みシンボル結果をフォーマットする。
func formatElixirSymbolResult(def genericSymbolDef, aliases, callers, behaviours, otherRefs, tests []genericSymbolRef, reg *locator.Registry) string {
	var sb strings.Builder

	header := fmt.Sprintf("── %s %s (L%d) in %s", def.Kind, def.Name, def.Line, def.File)
	if reg != nil {
		id := reg.Register(locator.Location{FilePath: def.File, Line: def.Line, Name: fmt.Sprintf("%s %s", def.Kind, def.Name)})
		header += " " + id
	}
	fmt.Fprintf(&sb, "%s ──\n", header)
	fmt.Fprintf(&sb, "%d: %s\n", def.Line, def.Signature)

	writeRefSection(&sb, "Aliases/Imports", aliases, exAliasLimit, reg)
	writeRefSection(&sb, "Callers", callers, exCallerLimit, reg)
	writeRefSection(&sb, "Behaviours", behaviours, exBehaviourLimit, reg)
	writeRefSection(&sb, "References", otherRefs, genericRefLimit, reg)
	writeRefSection(&sb, "Related Tests", tests, genericTestLimit, reg)

	total := len(aliases) + len(callers) + len(behaviours) + len(otherRefs) + len(tests)
	if total == 0 {
		sb.WriteString("\nNo references found.\n")
	}
	return sb.String()
}
