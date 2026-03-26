package search

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/locator"
)

const (
	javaImportLimit      = 5
	javaCallerLimit      = 10
	javaAnnotationLimit  = 5
	javaInheritanceLimit = 5
)

// resolveJavaSymbol は Java / Kotlin 向けの enhanced symbol fast path。
// 参照を import / caller / annotation / inheritance / other に分類する。
func resolveJavaSymbol(symbol string, opts SearchOptions) (string, genericSymbolStatus) {
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

	imports, callers, annotations, inheritance, otherRefs := classifyJavaRefs(normalRefs, symbol)
	return formatJavaSymbolResult(def, imports, callers, annotations, inheritance, otherRefs, testRefs, opts.LocatorRegistry), genericSymbolSingle
}

// classifyJavaRefs は Java/Kotlin の参照を分類する。
// 判定順: import → annotation → inheritance → caller → other。
func classifyJavaRefs(refs []genericSymbolRef, symbol string) (imports, callers, annotations, inheritance, others []genericSymbolRef) {
	escaped := regexp.QuoteMeta(symbol)
	importPat := regexp.MustCompile(`^\s*import\s+`)
	annotationPat := regexp.MustCompile(`@` + escaped + `\b`)
	inheritPat := regexp.MustCompile(`(?:extends|implements)\s+.*\b` + escaped + `\b|:\s*.*\b` + escaped + `\b`)
	callerPat := regexp.MustCompile(`\b` + escaped + `\s*[\(.]|\bnew\s+` + escaped + `\b`)

	for _, ref := range refs {
		s := ref.Snippet
		switch {
		case importPat.MatchString(s):
			imports = append(imports, ref)
		case annotationPat.MatchString(s):
			annotations = append(annotations, ref)
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

// formatJavaSymbolResult は Java/Kotlin の分類済みシンボル結果をフォーマットする。
func formatJavaSymbolResult(def genericSymbolDef, imports, callers, annotations, inheritance, otherRefs, tests []genericSymbolRef, reg *locator.Registry) string {
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

	writeRefSection(&sb, "Imports", imports, javaImportLimit, reg)
	writeRefSection(&sb, "Callers", callers, javaCallerLimit, reg)
	writeRefSection(&sb, "Annotations", annotations, javaAnnotationLimit, reg)
	writeRefSection(&sb, "Inheritance", inheritance, javaInheritanceLimit, reg)
	writeRefSection(&sb, "References", otherRefs, genericRefLimit, reg)
	writeRefSection(&sb, "Related Tests", tests, genericTestLimit, reg)

	total := len(imports) + len(callers) + len(annotations) + len(inheritance) + len(otherRefs) + len(tests)
	if total == 0 {
		sb.WriteString("\nNo references found.\n")
	}

	return sb.String()
}
