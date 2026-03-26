package search

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/locator"
)

const (
	pyImportLimit    = 5
	pyCallerLimit    = 10
	pyDecoratorLimit = 5
)

// resolvePythonSymbol は Python 向けの enhanced symbol fast path。
// 参照を import / caller / decorator / other に分類する。
func resolvePythonSymbol(symbol string, opts SearchOptions) (string, genericSymbolStatus) {
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

	imports, callers, decorators, otherRefs := classifyPythonRefs(normalRefs, symbol)
	return formatPythonSymbolResult(def, imports, callers, decorators, otherRefs, testRefs, opts.LocatorRegistry), genericSymbolSingle
}

// classifyPythonRefs は Python の参照を import / caller / decorator / other に分類する。
// 判定順: import → decorator → caller → other。
func classifyPythonRefs(refs []genericSymbolRef, symbol string) (imports, callers, decorators, others []genericSymbolRef) {
	escaped := regexp.QuoteMeta(symbol)
	importPat := regexp.MustCompile(`(?:^|\s)(?:import\s+` + escaped + `\b|from\s+.*\s+import\s+.*\b` + escaped + `\b|from\s+` + escaped + `[\s.])`)
	decoratorPat := regexp.MustCompile(`^\s*@` + escaped + `\b`)
	callerPat := regexp.MustCompile(`\b` + escaped + `\s*\(`)

	for _, ref := range refs {
		s := ref.Snippet
		switch {
		case importPat.MatchString(s):
			imports = append(imports, ref)
		case decoratorPat.MatchString(s):
			decorators = append(decorators, ref)
		case callerPat.MatchString(s):
			callers = append(callers, ref)
		default:
			others = append(others, ref)
		}
	}
	return
}

// formatPythonSymbolResult は Python の分類済みシンボル結果をフォーマットする。
func formatPythonSymbolResult(def genericSymbolDef, imports, callers, decorators, otherRefs, tests []genericSymbolRef, reg *locator.Registry) string {
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

	writeRefSection(&sb, "Imports", imports, pyImportLimit, reg)
	writeRefSection(&sb, "Callers", callers, pyCallerLimit, reg)
	writeRefSection(&sb, "Decorators", decorators, pyDecoratorLimit, reg)
	writeRefSection(&sb, "References", otherRefs, genericRefLimit, reg)
	writeRefSection(&sb, "Related Tests", tests, genericTestLimit, reg)

	total := len(imports) + len(callers) + len(decorators) + len(otherRefs) + len(tests)
	if total == 0 {
		sb.WriteString("\nNo references found.\n")
	}

	return sb.String()
}
