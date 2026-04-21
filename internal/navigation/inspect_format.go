package navigation

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/locator"
)

// formatMultipleCandidates は複数候補の一覧を整形する。
func formatMultipleCandidates(symbol string, candidates []SymbolCandidate, reg *locator.Registry) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Multiple symbols matched %q:\n", symbol)
	for i, c := range candidates {
		line := fmt.Sprintf(
			"  %d. %-40s %s %s (L%d-L%d)",
			i+1,
			c.File,
			c.Kind,
			candidateDisplayName(c),
			c.Line,
			c.EndLine,
		)
		if reg != nil {
			id := reg.Register(newInspectSymbolLocator(c.File, c.RootPath, c.Line, c.EndLine, fmt.Sprintf("%s %s", c.Kind, candidateDisplayName(c))))
			line += " " + id
		}
		fmt.Fprintf(&sb, "%s\n", line)
	}
	sb.WriteString("\nRefine with path or receiver-qualified symbol to disambiguate.")
	return sb.String()
}

// formatInspectResult は単一候補の結果を整形する。
// reg が nil でない場合、ヘッダーと参照行に Locator ID を付与する。
func formatInspectResult(r InspectResult, reg *locator.Registry) string {
	if r.Symbol == nil {
		return ""
	}

	writer := inspectResultWriter{
		result: r,
		symbol: *r.Symbol,
		reg:    reg,
	}
	writer.writeHeader()
	writer.writeBody()
	writer.writeCallers()
	writer.writeReferences()
	writer.writeRelatedTests()
	writer.writeImplementations()
	writer.writeUpstreamStatus()
	writer.writeLSPSummary()
	return writer.sb.String()
}

// inspectResultWriter は inspect 出力の section 整形責務を持つ。
type inspectResultWriter struct {
	result InspectResult
	symbol SymbolCandidate
	reg    *locator.Registry
	sb     strings.Builder
}

func (w *inspectResultWriter) writeHeader() {
	header := fmt.Sprintf(
		"── %s %s (L%d-L%d) in %s",
		w.symbol.Kind,
		candidateDisplayName(w.symbol),
		w.symbol.Line,
		w.symbol.EndLine,
		w.symbol.File,
	)
	if w.reg != nil {
		id := w.reg.Register(newInspectSymbolLocator(
			w.symbol.File,
			w.symbol.RootPath,
			w.symbol.Line,
			w.symbol.EndLine,
			fmt.Sprintf("%s %s", w.symbol.Kind, candidateDisplayName(w.symbol)),
		))
		header += " " + id
	}
	fmt.Fprintf(&w.sb, "%s ──\n", header)
}

func (w *inspectResultWriter) writeBody() {
	for _, line := range w.result.Body {
		w.sb.WriteString(line)
		w.sb.WriteByte('\n')
	}
}

func (w *inspectResultWriter) writeCallers() {
	if len(w.result.Callers) == 0 {
		return
	}
	w.writeRelatedHeader("Callers", len(w.result.Callers), w.result.TotalCallers, w.result.MoreCallers)
	for _, c := range w.result.Callers {
		scope := ""
		if c.Scope != "" && c.Scope != "package-level" {
			scope = " in " + c.Scope
		}
		line := fmt.Sprintf("  - %s:%d%s", c.File, c.Line, scope)
		line = w.appendRelatedLocator(line, c.File, c.ResolvedPath, c.Line, "")
		fmt.Fprintf(&w.sb, "%s\n", line)
	}
	if w.result.MoreCallers {
		w.sb.WriteString("  (+ more callers. Use search_code for more results)\n")
	}
}

func (w *inspectResultWriter) writeReferences() {
	if len(w.result.Refs) == 0 {
		return
	}
	w.writeRelatedHeader("References", len(w.result.Refs), w.result.TotalRefs, w.result.MoreRefs)
	for _, ref := range w.result.Refs {
		label := ""
		if ref.IsTest {
			label = " [test]"
		}
		line := fmt.Sprintf("  - %s:%d | %s%s", ref.File, ref.Line, strings.TrimSpace(ref.Snippet), label)
		line = w.appendRelatedLocator(line, ref.File, ref.ResolvedPath, ref.Line, "")
		fmt.Fprintf(&w.sb, "%s\n", line)
	}
	if w.result.MoreRefs {
		w.sb.WriteString("  (+ more references. Use search_code for more results)\n")
	}
}

func (w *inspectResultWriter) writeRelatedTests() {
	if len(w.result.Tests) == 0 {
		return
	}
	w.writeRelatedHeader("Related tests", len(w.result.Tests), w.result.TotalTests, w.result.MoreTests)
	for _, testRef := range w.result.Tests {
		line := fmt.Sprintf("  - %s:%d | func %s", testRef.File, testRef.Line, testRef.Name)
		line = w.appendRelatedLocator(line, testRef.File, testRef.ResolvedPath, testRef.Line, "func "+testRef.Name)
		fmt.Fprintf(&w.sb, "%s\n", line)
	}
	if w.result.MoreTests {
		w.sb.WriteString("  (+ more tests. Use search_code for more results)\n")
	}
}

func (w *inspectResultWriter) writeImplementations() {
	if len(w.result.Implementations) == 0 {
		return
	}
	fmt.Fprintf(&w.sb, "\nImplementations (%d):\n", len(w.result.Implementations))
	for _, impl := range w.result.Implementations {
		line := fmt.Sprintf("  - %s:%d %s", impl.File, impl.Line, impl.Name)
		line = w.appendRelatedLocator(line, impl.File, impl.ResolvedPath, impl.Line, "")
		fmt.Fprintf(&w.sb, "%s\n", line)
	}
}

func (w *inspectResultWriter) writeUpstreamStatus() {
	if w.result.UpstreamIncomplete {
		w.sb.WriteString("\nWarning: Upstream search may be incomplete due to errors. Use narrower search scopes.\n")
		return
	}
	if w.result.UpstreamTruncated {
		w.sb.WriteString("\nNote: Some results were truncated upstream. For comprehensive search, use search_code.\n")
	}
}

func (w *inspectResultWriter) writeLSPSummary() {
	if !w.result.ResolvedViaLSP {
		return
	}
	fmt.Fprintf(&w.sb, "\n(resolved via gopls · %d callers, %d refs", w.result.TotalCallers, w.result.TotalRefs)
	if len(w.result.Implementations) > 0 {
		fmt.Fprintf(&w.sb, ", %d impls", len(w.result.Implementations))
	}
	w.sb.WriteString(")\n")
}

func (w *inspectResultWriter) writeRelatedHeader(title string, shown, total int, more bool) {
	if more {
		fmt.Fprintf(&w.sb, "\n%s: %d examples (of %d found)\n", title, shown, total)
		return
	}
	fmt.Fprintf(&w.sb, "\n%s (%d):\n", title, shown)
}

func (w *inspectResultWriter) appendRelatedLocator(line, filePath, resolvedPath string, lineNo int, name string) string {
	if w.reg == nil {
		return line
	}
	id := w.reg.Register(newInspectRelatedLocator(filePath, resolvedPath, w.symbol.RootPath, lineNo, 0, name))
	return line + " " + id
}

func newInspectSymbolLocator(filePath, rootPath string, line, endLine int, name string) locator.Location {
	return locator.Location{
		FilePath:     filePath,
		ResolvedPath: resolveInspectLocatorPath(filePath, rootPath),
		Line:         line,
		EndLine:      endLine,
		Name:         name,
	}
}

func newInspectRelatedLocator(filePath, resolvedPath, rootPath string, line, endLine int, name string) locator.Location {
	if strings.TrimSpace(resolvedPath) == "" {
		resolvedPath = resolveInspectLocatorPath(filePath, rootPath)
	} else {
		resolvedPath = cleanInspectResolvedPath(resolvedPath)
	}
	return locator.Location{
		FilePath:     filePath,
		ResolvedPath: resolvedPath,
		Line:         line,
		EndLine:      endLine,
		Name:         name,
	}
}

func resolveInspectLocatorPath(filePath, rootPath string) string {
	filePath = strings.TrimSpace(filePath)
	if filePath == "" {
		return ""
	}
	if filepath.IsAbs(filePath) {
		return filepath.Clean(filePath)
	}
	rootPath = strings.TrimSpace(rootPath)
	if rootPath == "" {
		return ""
	}
	if abs, err := filepath.Abs(rootPath); err == nil {
		rootPath = abs
	}
	return filepath.Clean(filepath.Join(rootPath, filepath.FromSlash(filePath)))
}
