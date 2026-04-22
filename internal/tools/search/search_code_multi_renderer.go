package search

import (
	"fmt"
	"strings"
)

type patternSymbolBundleGroup struct {
	Bundle   *SymbolBundle
	Patterns []string
}

type patternSymbolBundleGroupCollector struct {
	groups map[string]patternSymbolBundleGroup
}

func renderMultiPatternOutput(collected []formattedPatternExecution, patternCount int, opts SearchOptions) string {
	var sb strings.Builder
	grouped := groupPatternSymbolBundles(collected)
	emittedBundles := make(map[string]bool)
	for i, execution := range collected {
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		appendMultiPatternExecutionOutput(&sb, grouped, emittedBundles, execution, i, patternCount, opts)
	}

	if idx := buildCrossPatternIndexFromExecutions(collected, opts.LocatorRegistry, opts); idx != "" {
		sb.WriteString(idx)
	}
	return sb.String() + lineRangeHint
}

func appendMultiPatternExecutionOutput(sb *strings.Builder, grouped map[string]patternSymbolBundleGroup, emittedBundles map[string]bool, execution formattedPatternExecution, patternIndex, patternCount int, opts SearchOptions) {
	if wroteBundle := appendGroupedSymbolBundleOutput(sb, grouped, emittedBundles, execution, opts); wroteBundle {
		return
	}
	appendPlainPatternExecutionOutput(sb, execution, patternIndex, patternCount)
}

func appendPlainPatternExecutionOutput(sb *strings.Builder, execution formattedPatternExecution, patternIndex, patternCount int) {
	fmt.Fprintf(sb, "━━ Pattern %d/%d: %q ━━\n", patternIndex+1, patternCount, execution.Pattern)
	appendRenderedSection(sb, execution.Output)
}

func appendGroupedSymbolBundleOutput(sb *strings.Builder, grouped map[string]patternSymbolBundleGroup, emittedBundles map[string]bool, execution formattedPatternExecution, opts SearchOptions) bool {
	if execution.Bundle == nil {
		return false
	}
	canonical := execution.Bundle.Identity.Canonical
	group, ok := grouped[canonical]
	if !ok || len(group.Patterns) <= 1 {
		return false
	}
	if emittedBundles[canonical] {
		return true
	}

	emittedBundles[canonical] = true

	fmt.Fprintf(sb, "━━ Symbol Bundle: %q ━━\n", group.Bundle.Identity.DisplayName)
	appendRenderedSection(sb, formatSymbolBundle(group.Bundle, opts.LocatorRegistry, group.Patterns))
	return true
}

func appendRenderedSection(sb *strings.Builder, section string) {
	sb.WriteString(section)
	if !strings.HasSuffix(section, "\n") {
		sb.WriteString("\n")
	}
}

func groupPatternSymbolBundles(collected []formattedPatternExecution) map[string]patternSymbolBundleGroup {
	collector := newPatternSymbolBundleGroupCollector()
	for _, item := range collected {
		collector.addExecution(item)
	}
	return collector.groups
}

func newPatternSymbolBundleGroupCollector() *patternSymbolBundleGroupCollector {
	return &patternSymbolBundleGroupCollector{
		groups: make(map[string]patternSymbolBundleGroup),
	}
}

func (collector *patternSymbolBundleGroupCollector) addExecution(execution formattedPatternExecution) {
	if execution.Bundle == nil {
		return
	}
	key := execution.Bundle.Identity.Canonical
	group := collector.groups[key]
	if group.Bundle == nil {
		group.Bundle = execution.Bundle
	}
	group.Patterns = appendPatternSymbolBundleCandidates(group.Patterns, execution)
	collector.groups[key] = group
}

func appendPatternSymbolBundleCandidates(patterns []string, execution formattedPatternExecution) []string {
	for _, candidate := range patternSymbolBundleCandidates(execution) {
		patterns = appendPatternIfMissing(patterns, candidate)
	}
	return patterns
}

func patternSymbolBundleCandidates(execution formattedPatternExecution) []string {
	candidates := make([]string, 0, 1+len(execution.Route.SymbolCandidates))
	candidates = append(candidates, execution.Pattern)
	candidates = append(candidates, execution.Route.SymbolCandidates...)
	return candidates
}
