package search

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/navigation"
)

func supplementGoImpactTestsFromProbe(symbol string, result navigation.InspectResult, opts SearchOptions, limit int) (navigation.InspectResult, []string) {
	if limit <= 0 || result.Symbol == nil || len(result.Tests) > 0 || result.TotalTests > 0 {
		return result, nil
	}

	probe := structuredGoImpactTestProbePattern(symbol, *result.Symbol)
	if strings.TrimSpace(probe) == "" {
		return result, nil
	}

	tests, total, dependencies := findStructuredGoImpactTestsByProbe(probe, *result.Symbol, opts, limit)
	if len(tests) == 0 {
		return result, dependencies
	}

	result.Tests = tests
	result.TotalTests = total
	result.MoreTests = total > len(tests)
	return result, dependencies
}

func structuredGoImpactTestProbePattern(rawSymbol string, symbol navigation.SymbolCandidate) string {
	if name := strings.TrimSpace(symbol.Name); name != "" {
		return impactTestProbePattern(name)
	}
	return impactTestProbePattern(rawSymbol)
}

func findStructuredGoImpactTestsByProbe(probe string, symbol navigation.SymbolCandidate, opts SearchOptions, limit int) ([]navigation.TestRef, int, []string) {
	if symbol.Kind == "method" && strings.TrimSpace(symbol.Receiver) != "" {
		if structuredGoImpactMethodProbePathIsFile(opts) {
			tests, total := findGoImpactTestsByNameProbe(probe, symbol.RootPath, opts, limit)
			return tests, total, nil
		}
		return findGoImpactMethodTestsByNameProbe(probe, symbol, opts, limit)
	}
	tests, total := findGoImpactTestsByNameProbe(probe, symbol.RootPath, opts, limit)
	return tests, total, nil
}

func findGoImpactMethodTestsByNameProbe(probe string, symbol navigation.SymbolCandidate, opts SearchOptions, limit int) ([]navigation.TestRef, int, []string) {
	packageDir := structuredGoImpactMethodProbeDir(symbol, opts)
	if packageDir == "" {
		return nil, 0, nil
	}

	entries, err := os.ReadDir(packageDir)
	if err != nil {
		return nil, 0, nil
	}

	deps := newStructuredGoImpactProbeDeps()
	ctx, ok := newGoMethodTestProbeContext(probe, symbol, opts, packageDir, entries, deps)
	if !ok {
		return nil, 0, deps.list()
	}

	tests := make([]navigation.TestRef, 0, min(limit, len(entries)))
	total := 0
	seen := make(map[string]struct{})
	appendTest := func(test navigation.TestRef) {
		key := fmt.Sprintf("%s:%d:%s", test.File, test.Line, test.Name)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		total++
		if len(tests) < limit {
			tests = append(tests, test)
		}
	}

	ctx.collectLocalTests(entries, appendTest)
	ctx.collectCrossPackageTests(appendTest)
	return tests, total, deps.list()
}

func (ctx goMethodTestProbeContext) collectCrossPackageTests(appendTest func(navigation.TestRef)) {
	if !ctx.symbol.Exported {
		return
	}

	matcher := newGoMethodCrossPackageTestMatcher(ctx)
	broader, _ := findGoImpactTestsByNameProbe(ctx.probe, ctx.symbol.RootPath, ctx.opts, 0)
	for _, candidate := range broader {
		absPath := absoluteAffectedFilePathWithPreferredBases(
			candidate.File,
			ctx.symbol.RootPath,
			affectedFileBasePath(ctx.opts, affectedFileSourceText),
			structuredGoImpactProbeRootPath(ctx.opts, ctx.packageDir),
		)
		ctx.dependencies.add(absPath)
		if absPath == "" || filepath.Clean(filepath.Dir(absPath)) == filepath.Clean(ctx.packageDir) {
			continue
		}
		if !matcher.matches(absPath, candidate) {
			continue
		}
		appendTest(candidate)
	}
}
