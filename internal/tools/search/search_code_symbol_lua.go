package search

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/locator"
)

const (
	luaRequireLimit = 5
	luaCallerLimit  = 10
)

// resolveLuaSymbol は Lua 向けの enhanced symbol fast path。
func resolveLuaSymbol(symbol string, opts SearchOptions) (string, genericSymbolStatus) {
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

	requires, callers, otherRefs := classifyLuaRefs(normalRefs, symbol)
	return formatLuaSymbolResult(def, requires, callers, otherRefs, testRefs, opts.LocatorRegistry), genericSymbolSingle
}

// classifyLuaRefs は Lua の参照を分類する。
func classifyLuaRefs(refs []genericSymbolRef, symbol string) (requires, callers, others []genericSymbolRef) {
	escaped := regexp.QuoteMeta(symbol)
	requirePat := regexp.MustCompile(`\brequire\s*[\("]`)
	callerPat := regexp.MustCompile(`\b` + escaped + `\s*[\(.:]\b?`)

	for _, ref := range refs {
		s := ref.Snippet
		switch {
		case requirePat.MatchString(s):
			requires = append(requires, ref)
		case callerPat.MatchString(s):
			callers = append(callers, ref)
		default:
			others = append(others, ref)
		}
	}
	return
}

// formatLuaSymbolResult は Lua の分類済みシンボル結果をフォーマットする。
func formatLuaSymbolResult(def genericSymbolDef, requires, callers, otherRefs, tests []genericSymbolRef, reg *locator.Registry) string {
	var sb strings.Builder

	header := fmt.Sprintf("── %s %s (L%d) in %s", def.Kind, def.Name, def.Line, def.File)
	if reg != nil {
		id := reg.Register(locator.Location{FilePath: def.File, Line: def.Line, Name: fmt.Sprintf("%s %s", def.Kind, def.Name)})
		header += " " + id
	}
	fmt.Fprintf(&sb, "%s ──\n", header)
	fmt.Fprintf(&sb, "%d: %s\n", def.Line, def.Signature)

	writeRefSection(&sb, "Requires", requires, luaRequireLimit, reg)
	writeRefSection(&sb, "Callers", callers, luaCallerLimit, reg)
	writeRefSection(&sb, "References", otherRefs, genericRefLimit, reg)
	writeRefSection(&sb, "Related Tests", tests, genericTestLimit, reg)

	total := len(requires) + len(callers) + len(otherRefs) + len(tests)
	if total == 0 {
		sb.WriteString("\nNo references found.\n")
	}
	return sb.String()
}
