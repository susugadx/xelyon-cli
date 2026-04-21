package search

import (
	"fmt"
	"strings"
)

type patternSymbolBundleGroup struct {
	Bundle   *SymbolBundle
	Patterns []string
	Emitted  bool
}

func renderMultiPatternOutput(collected []formattedPatternExecution, patterns []string, opts SearchOptions) string {
	var sb strings.Builder
	grouped := groupPatternSymbolBundles(collected)
	for i, execution := range collected {
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		appendMultiPatternExecutionOutput(&sb, grouped, execution, i, len(patterns), opts)
	}

	if idx := buildCrossPatternIndexFromExecutions(collected, opts.LocatorRegistry, opts); idx != "" {
		sb.WriteString(idx)
	}
	return sb.String() + lineRangeHint
}

func appendMultiPatternExecutionOutput(sb *strings.Builder, grouped map[string]patternSymbolBundleGroup, execution formattedPatternExecution, patternIndex, patternCount int, opts SearchOptions) {
	if wroteBundle := appendGroupedSymbolBundleOutput(sb, grouped, execution, opts); wroteBundle {
		return
	}
	appendPlainPatternExecutionOutput(sb, execution, patternIndex, patternCount)
}

func appendPlainPatternExecutionOutput(sb *strings.Builder, execution formattedPatternExecution, patternIndex, patternCount int) {
	fmt.Fprintf(sb, "━━ Pattern %d/%d: %q ━━\n", patternIndex+1, patternCount, execution.Pattern)
	sb.WriteString(execution.Output)
	if !strings.HasSuffix(execution.Output, "\n") {
		sb.WriteString("\n")
	}
}

func appendGroupedSymbolBundleOutput(sb *strings.Builder, grouped map[string]patternSymbolBundleGroup, execution formattedPatternExecution, opts SearchOptions) bool {
	if execution.Bundle == nil {
		return false
	}
	group, ok := grouped[execution.Bundle.Identity.Canonical]
	if !ok || len(group.Patterns) <= 1 {
		return false
	}
	if group.Emitted {
		return true
	}

	group.Emitted = true
	grouped[execution.Bundle.Identity.Canonical] = group

	fmt.Fprintf(sb, "━━ Symbol Bundle: %q ━━\n", group.Bundle.Identity.DisplayName)
	sb.WriteString(formatSymbolBundle(group.Bundle, opts.LocatorRegistry, group.Patterns))
	if !strings.HasSuffix(sb.String(), "\n") {
		sb.WriteString("\n")
	}
	return true
}

func groupPatternSymbolBundles(collected []formattedPatternExecution) map[string]patternSymbolBundleGroup {
	groups := make(map[string]patternSymbolBundleGroup)
	for _, item := range collected {
		if item.Bundle == nil {
			continue
		}
		key := item.Bundle.Identity.Canonical
		group := groups[key]
		if group.Bundle == nil {
			group.Bundle = item.Bundle
		}
		group.Patterns = appendPatternIfMissing(group.Patterns, item.Pattern)
		for _, candidate := range item.Route.SymbolCandidates {
			group.Patterns = appendPatternIfMissing(group.Patterns, candidate)
		}
		groups[key] = group
	}
	return groups
}
