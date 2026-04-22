package navigation

import (
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/locator"
)

// inspectLocatorDecorator は inspect 出力に locator id を付与する責務を持つ。
type inspectLocatorDecorator struct {
	symbol SymbolCandidate
	reg    *locator.Registry
}

func newInspectLocatorDecorator(symbol SymbolCandidate, reg *locator.Registry) inspectLocatorDecorator {
	return inspectLocatorDecorator{symbol: symbol, reg: reg}
}

func (d inspectLocatorDecorator) decorateCandidateLine(line string, candidate SymbolCandidate) string {
	if d.reg == nil {
		return line
	}
	id := d.reg.Register(newInspectSymbolLocator(
		candidate.File,
		candidate.RootPath,
		candidate.Line,
		candidate.EndLine,
		fmt.Sprintf("%s %s", candidate.Kind, candidateDisplayName(candidate)),
	))
	return line + " " + id
}

func (d inspectLocatorDecorator) decorateHeaderLine(line string) string {
	if d.reg == nil {
		return line
	}
	id := d.reg.Register(newInspectSymbolLocator(
		d.symbol.File,
		d.symbol.RootPath,
		d.symbol.Line,
		d.symbol.EndLine,
		fmt.Sprintf("%s %s", d.symbol.Kind, candidateDisplayName(d.symbol)),
	))
	return line + " " + id
}

func (d inspectLocatorDecorator) decorateRelatedLine(line, filePath, resolvedPath string, lineNo int, name string) string {
	if d.reg == nil {
		return line
	}
	id := d.reg.Register(newInspectRelatedLocator(filePath, resolvedPath, d.symbol.RootPath, lineNo, 0, name))
	return line + " " + id
}
