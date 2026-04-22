package navigation

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/locator"
)

// formatMultipleCandidates は複数候補の一覧を整形する。
func formatMultipleCandidates(symbol string, candidates []SymbolCandidate, reg *locator.Registry) string {
	var sb strings.Builder
	decorator := newInspectLocatorDecorator(SymbolCandidate{}, reg)
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
		line = decorator.decorateCandidateLine(line, c)
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

	symbol := *r.Symbol
	writer := inspectResultWriter{
		result:    r,
		symbol:    symbol,
		decorator: newInspectLocatorDecorator(symbol, reg),
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
	result    InspectResult
	symbol    SymbolCandidate
	decorator inspectLocatorDecorator
	sb        strings.Builder
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
	header = w.decorator.decorateHeaderLine(header)
	fmt.Fprintf(&w.sb, "%s ──\n", header)
}

func (w *inspectResultWriter) writeBody() {
	for _, line := range w.result.Body {
		w.sb.WriteString(line)
		w.sb.WriteByte('\n')
	}
}
