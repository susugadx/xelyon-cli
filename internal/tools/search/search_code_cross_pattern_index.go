package search

import (
	"path/filepath"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/locator"
)

func parseNumberedCandidateFilePath(line string) (string, bool) {
	line = strings.TrimSpace(line)
	if line == "" {
		return "", false
	}
	dotIdx := strings.Index(line, ".")
	if dotIdx <= 0 {
		return "", false
	}
	for _, r := range line[:dotIdx] {
		if r < '0' || r > '9' {
			return "", false
		}
	}
	rest := strings.TrimSpace(line[dotIdx+1:])
	if rest == "" {
		return "", false
	}
	if idx := strings.Index(rest, " function "); idx > 0 {
		return strings.TrimSpace(rest[:idx]), true
	}
	if idx := strings.Index(rest, " method "); idx > 0 {
		return strings.TrimSpace(rest[:idx]), true
	}
	if idx := strings.Index(rest, " type "); idx > 0 {
		return strings.TrimSpace(rest[:idx]), true
	}
	if idx := strings.Index(rest, " interface "); idx > 0 {
		return strings.TrimSpace(rest[:idx]), true
	}
	if idx := strings.Index(rest, " const "); idx > 0 {
		return strings.TrimSpace(rest[:idx]), true
	}
	if idx := strings.Index(rest, " var "); idx > 0 {
		return strings.TrimSpace(rest[:idx]), true
	}
	return "", false
}

func hasNumericListPrefix(line string) bool {
	line = strings.TrimSpace(line)
	if line == "" {
		return false
	}
	dotIdx := strings.Index(line, ".")
	if dotIdx <= 0 {
		return false
	}
	for _, r := range line[:dotIdx] {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

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
	return buildCrossPatternIndexFromExecutions(collected, reg, opts)
}

type crossPatternIndexEntry struct {
	ref          primaryFileRef
	patternCount int
	category     string
}

type crossPatternIndexSections struct {
	implKeys   []string
	testKeys   []string
	configKeys []string
}

func buildCrossPatternIndexFromExecutions(collected []formattedPatternExecution, reg *locator.Registry, opts SearchOptions) string {
	fileMap, order := collectCrossPatternIndexEntries(collected, opts)
	if len(order) == 0 {
		return ""
	}

	sections := splitCrossPatternIndexSections(fileMap, order)
	hasHotspot := hasCrossPatternHotspot(fileMap)
	if !shouldRenderCrossPatternIndex(order, sections, hasHotspot) {
		return ""
	}
	return renderCrossPatternIndex(fileMap, order, sections, reg)
}

func collectCrossPatternIndexEntries(collected []formattedPatternExecution, opts SearchOptions) (map[string]*crossPatternIndexEntry, []string) {
	fileMap := make(map[string]*crossPatternIndexEntry)
	var order []string
	for _, execution := range collected {
		for _, ref := range primaryFileRefsFromExecution(execution, opts) {
			key := ref.DisplayPath + "\x00" + ref.ResolvedPath
			if entry, ok := fileMap[key]; ok {
				entry.patternCount++
				continue
			}
			fileMap[key] = &crossPatternIndexEntry{
				ref:          ref,
				patternCount: 1,
				category:     classifyFilePath(ref.DisplayPath),
			}
			order = append(order, key)
		}
	}
	return fileMap, order
}

func splitCrossPatternIndexSections(fileMap map[string]*crossPatternIndexEntry, order []string) crossPatternIndexSections {
	sections := crossPatternIndexSections{}
	for _, key := range order {
		switch fileMap[key].category {
		case "test":
			sections.testKeys = append(sections.testKeys, key)
		case "config":
			sections.configKeys = append(sections.configKeys, key)
		default:
			sections.implKeys = append(sections.implKeys, key)
		}
	}
	return sections
}

func hasCrossPatternHotspot(fileMap map[string]*crossPatternIndexEntry) bool {
	for _, entry := range fileMap {
		if entry.patternCount > 1 {
			return true
		}
	}
	return false
}

func shouldRenderCrossPatternIndex(order []string, sections crossPatternIndexSections, hasHotspot bool) bool {
	categoryCount := 0
	if len(sections.implKeys) > 0 {
		categoryCount++
	}
	if len(sections.testKeys) > 0 {
		categoryCount++
	}
	if len(sections.configKeys) > 0 {
		categoryCount++
	}
	return hasHotspot || categoryCount >= 2 || len(order) >= 3
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
