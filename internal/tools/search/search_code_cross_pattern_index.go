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

type crossPatternIndexCollector struct {
	fileMap map[string]*crossPatternIndexEntry
	order   []string
}

type crossPatternIndexData struct {
	fileMap    map[string]*crossPatternIndexEntry
	order      []string
	sections   crossPatternIndexSections
	hasHotspot bool
}

type crossPatternIndexRenderPolicy struct {
	MinCategoryCount int
	MinUniqueFiles   int
}

var defaultCrossPatternIndexRenderPolicy = crossPatternIndexRenderPolicy{
	MinCategoryCount: 2,
	MinUniqueFiles:   3,
}

func buildCrossPatternIndexFromExecutions(collected []formattedPatternExecution, reg *locator.Registry, opts SearchOptions) string {
	data := buildCrossPatternIndexData(collected, opts)
	if data.isEmpty() {
		return ""
	}
	if !data.shouldRender() {
		return ""
	}
	return renderCrossPatternIndex(data.fileMap, data.order, data.sections, reg)
}

func buildCrossPatternIndexData(collected []formattedPatternExecution, opts SearchOptions) crossPatternIndexData {
	fileMap, order := collectCrossPatternIndexEntries(collected, opts)
	sections, hasHotspot := summarizeCrossPatternIndex(fileMap, order)
	return crossPatternIndexData{
		fileMap:    fileMap,
		order:      order,
		sections:   sections,
		hasHotspot: hasHotspot,
	}
}

func (d crossPatternIndexData) isEmpty() bool {
	return len(d.order) == 0
}

func (d crossPatternIndexData) shouldRender() bool {
	return shouldRenderCrossPatternIndex(d.order, d.sections, d.hasHotspot)
}

func collectCrossPatternIndexEntries(collected []formattedPatternExecution, opts SearchOptions) (map[string]*crossPatternIndexEntry, []string) {
	collector := newCrossPatternIndexCollector()
	for _, execution := range collected {
		for _, ref := range primaryFileRefsFromExecution(execution, opts) {
			collector.addRef(ref)
		}
	}
	return collector.results()
}

func crossPatternIndexEntryKey(ref primaryFileRef) string {
	return ref.DisplayPath + "\x00" + ref.ResolvedPath
}

func newCrossPatternIndexCollector() *crossPatternIndexCollector {
	return &crossPatternIndexCollector{
		fileMap: make(map[string]*crossPatternIndexEntry),
	}
}

func (collector *crossPatternIndexCollector) addRef(ref primaryFileRef) {
	key := crossPatternIndexEntryKey(ref)
	if entry, ok := collector.fileMap[key]; ok {
		entry.patternCount++
		return
	}
	collector.fileMap[key] = &crossPatternIndexEntry{
		ref:          ref,
		patternCount: 1,
		category:     classifyFilePath(ref.DisplayPath),
	}
	collector.order = append(collector.order, key)
}

func (collector *crossPatternIndexCollector) results() (map[string]*crossPatternIndexEntry, []string) {
	return collector.fileMap, collector.order
}

func summarizeCrossPatternIndex(fileMap map[string]*crossPatternIndexEntry, order []string) (crossPatternIndexSections, bool) {
	sections := crossPatternIndexSections{}
	hasHotspot := false
	for _, key := range order {
		entry := fileMap[key]
		if entry.patternCount > 1 {
			hasHotspot = true
		}
		switch entry.category {
		case "test":
			sections.testKeys = append(sections.testKeys, key)
		case "config":
			sections.configKeys = append(sections.configKeys, key)
		default:
			sections.implKeys = append(sections.implKeys, key)
		}
	}
	return sections, hasHotspot
}

func shouldRenderCrossPatternIndex(order []string, sections crossPatternIndexSections, hasHotspot bool) bool {
	return defaultCrossPatternIndexRenderPolicy.shouldRender(order, sections, hasHotspot)
}

func (sections crossPatternIndexSections) categoryCount() int {
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
	return categoryCount
}

func (policy crossPatternIndexRenderPolicy) shouldRender(order []string, sections crossPatternIndexSections, hasHotspot bool) bool {
	if hasHotspot {
		return true
	}
	if sections.categoryCount() >= policy.MinCategoryCount {
		return true
	}
	return len(order) >= policy.MinUniqueFiles
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
