package navigation

import (
	"fmt"
	"strings"
)

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
		line = w.decorator.decorateRelatedLine(line, c.File, c.ResolvedPath, c.Line, "")
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
		line = w.decorator.decorateRelatedLine(line, ref.File, ref.ResolvedPath, ref.Line, "")
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
		line = w.decorator.decorateRelatedLine(line, testRef.File, testRef.ResolvedPath, testRef.Line, "func "+testRef.Name)
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
		line = w.decorator.decorateRelatedLine(line, impl.File, impl.ResolvedPath, impl.Line, "")
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
