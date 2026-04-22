package search

import (
	"path/filepath"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/locator"
)

func classifyFilePath(path string) string {
	base := filepath.Base(path)
	if strings.HasSuffix(base, "_test.go") || strings.Contains(base, ".test.") ||
		strings.Contains(base, ".spec.") || strings.HasPrefix(base, "test_") {
		return "test"
	}
	switch filepath.Ext(base) {
	case ".yaml", ".yml", ".toml", ".env", ".ini", ".cfg", ".conf":
		return "config"
	}
	return "impl"
}

func buildCrossPatternIndex(patterns, outputs []string, reg *locator.Registry) string {
	return buildCrossPatternIndexWithOptions(patterns, outputs, reg, SearchOptions{})
}

func buildCrossPatternIndexWithOptions(patterns, outputs []string, reg *locator.Registry, opts SearchOptions) string {
	return buildCrossPatternIndexFromExecutions(buildCrossPatternExecutions(patterns, outputs), reg, opts)
}

func buildCrossPatternExecutions(patterns, outputs []string) []formattedPatternExecution {
	collected := make([]formattedPatternExecution, 0, min(len(patterns), len(outputs)))
	for i, output := range outputs {
		if i >= len(patterns) {
			break
		}
		collected = append(collected, formattedPatternExecution{
			Index: i,
			singlePatternExecution: singlePatternExecution{
				Pattern: patterns[i],
				Output:  output,
			},
		})
	}
	return collected
}

func buildCrossPatternIndexFromExecutions(collected []formattedPatternExecution, reg *locator.Registry, opts SearchOptions) string {
	data := buildCrossPatternIndexData(collected, opts)
	if data.isEmpty() {
		return ""
	}
	if !shouldRenderCrossPatternIndexData(data) {
		return ""
	}
	return renderCrossPatternIndex(data.fileMap, data.order, data.sections, reg)
}

func buildCrossPatternIndexData(collected []formattedPatternExecution, opts SearchOptions) crossPatternIndexData {
	collector := newCrossPatternIndexCollector()
	for _, execution := range collected {
		for _, ref := range primaryFileRefsFromExecution(execution, opts) {
			collector.addRef(ref)
		}
	}
	return newCrossPatternIndexData(collector)
}

func primaryFileRefsFromExecution(execution formattedPatternExecution, opts SearchOptions) []primaryFileRef {
	return primaryFileRefsFromBundleOrOutput(execution.Bundle, execution.Output, opts)
}

func primaryFileRefsFromBundleOrOutput(bundle *SymbolBundle, output string, opts SearchOptions) []primaryFileRef {
	if ref, ok := primaryFileRefFromBundle(bundle); ok {
		return []primaryFileRef{ref}
	}
	return extractPrimaryFileRefs(output, opts)
}

func primaryFileRefFromBundle(bundle *SymbolBundle) (primaryFileRef, bool) {
	if bundle == nil {
		return primaryFileRef{}, false
	}
	displayPath := strings.TrimSpace(bundle.Identity.File)
	if displayPath == "" {
		return primaryFileRef{}, false
	}
	return primaryFileRef{
		DisplayPath:  displayPath,
		ResolvedPath: cleanResolvedLocatorPath(absoluteAffectedFilePathWithBase(displayPath, bundle.Debug.FileRootPath)),
	}, true
}
