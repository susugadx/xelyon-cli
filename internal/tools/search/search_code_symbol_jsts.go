package search

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/locator"
)

const (
	jsImportLimit  = 5
	jsCallerLimit  = 10
	jsTypeRefLimit = 5
)

// resolveJSSymbol は TS/JS 向けの enhanced symbol fast path。
// 参照を import / caller / type ref / other に分類し、Go fast path に近い構造化結果を返す。
// フォールバック: 定義が見つからなければ genericSymbolNone を返し、text search に委譲する。
func resolveJSSymbol(symbol string, opts SearchOptions) (string, genericSymbolStatus) {
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

	imports, callers, typeRefs, otherRefs := classifyJSRefs(normalRefs, symbol)
	return formatJSSymbolResult(def, imports, callers, typeRefs, otherRefs, testRefs, opts.LocatorRegistry), genericSymbolSingle
}

// classifyJSRefs は TS/JS の参照を import / caller / type ref / other に分類する。
// 判定順: import → caller → type ref → other（最初にマッチした分類に入る）。
func classifyJSRefs(refs []genericSymbolRef, symbol string) (imports, callers, typeRefs, others []genericSymbolRef) {
	escaped := regexp.QuoteMeta(symbol)
	importPat := regexp.MustCompile(`(?:^|\s)(?:import\s|from\s|require\s*\(|export\s+.*\sfrom\s)`)
	callerPat := regexp.MustCompile(`(?:\b` + escaped + `\s*\(|\bnew\s+` + escaped + `\b)`)
	typePat := regexp.MustCompile(`(?::\s*` + escaped + `\b|<` + escaped + `\b|extends\s+` + escaped + `\b|implements\s+` + escaped + `\b)`)

	for _, ref := range refs {
		s := ref.Snippet
		switch {
		case importPat.MatchString(s):
			imports = append(imports, ref)
		case callerPat.MatchString(s):
			callers = append(callers, ref)
		case typePat.MatchString(s):
			typeRefs = append(typeRefs, ref)
		default:
			others = append(others, ref)
		}
	}
	return
}

// formatJSSymbolResult は TS/JS の分類済みシンボル結果をフォーマットする。
// 出力形式は Go fast path と同様のセクション構造で、locator ID を付与する。
func formatJSSymbolResult(def genericSymbolDef, imports, callers, typeRefs, otherRefs, tests []genericSymbolRef, reg *locator.Registry) string {
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

	writeRefSection(&sb, "Imports", imports, jsImportLimit, reg)
	writeRefSection(&sb, "Callers", callers, jsCallerLimit, reg)
	writeRefSection(&sb, "Type References", typeRefs, jsTypeRefLimit, reg)
	writeRefSection(&sb, "References", otherRefs, genericRefLimit, reg)
	writeRefSection(&sb, "Related Tests", tests, genericTestLimit, reg)

	total := len(imports) + len(callers) + len(typeRefs) + len(otherRefs) + len(tests)
	if total == 0 {
		sb.WriteString("\nNo references found.\n")
	}

	return sb.String()
}

// writeRefSection は参照セクションを書き出す共通ヘルパー。
// Python / Rust 拡張時にも再利用できる。
func writeRefSection(sb *strings.Builder, title string, refs []genericSymbolRef, limit int, reg *locator.Registry) {
	if len(refs) == 0 {
		return
	}
	if len(refs) > limit {
		fmt.Fprintf(sb, "\n── %s: %d shown (of %d) ──\n", title, limit, len(refs))
	} else {
		fmt.Fprintf(sb, "\n── %s (%d) ──\n", title, len(refs))
	}
	for i, ref := range refs {
		if i >= limit {
			fmt.Fprintf(sb, "  ... (%d more)\n", len(refs)-limit)
			break
		}
		line := fmt.Sprintf("  %s:%d  %s", ref.File, ref.Line, ref.Snippet)
		if reg != nil {
			id := reg.Register(locator.Location{FilePath: ref.File, Line: ref.Line})
			line += " " + id
		}
		fmt.Fprintf(sb, "%s\n", line)
	}
}
