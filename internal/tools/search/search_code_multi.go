package search

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

// patternResult は複数パターン検索の各パターンの結果
type patternResult struct {
	Pattern      string
	Results      []SearchResult
	Truncated    bool
	Index        int
	TotalMatches int // truncate前の全マッチ数（バジェット比例配分に使用）
	Error        string
	Warnings     []string
}

type formattedPatternExecution struct {
	Index int
	singlePatternExecution
}

// executeMultiplePatterns は複数パターンを goroutine 並列で検索する。
// 各パターンは executeSinglePattern に委譲し、symbol fast path + キャッシュが自動で効く。
func executeMultiplePatterns(cache tools.ToolCacheInterface, patterns []string, opts SearchOptions) string {
	collected := collectMultiPatternExecutions(cache, patterns, opts)
	output := renderMultiPatternOutput(collected, patterns, opts)
	writeMultiPatternCache(cache, patterns, opts, output, collected)
	return output
}

func collectMultiPatternExecutions(cache tools.ToolCacheInterface, patterns []string, opts SearchOptions) []formattedPatternExecution {
	ch := make(chan formattedPatternExecution, len(patterns))
	for i, p := range patterns {
		go func(idx int, pat string) {
			result := executeSinglePatternDetailed(cache, pat, opts)
			result.Output = strings.TrimSuffix(result.Output, lineRangeHint)
			ch <- formattedPatternExecution{Index: idx, singlePatternExecution: result}
		}(i, p)
	}

	collected := make([]formattedPatternExecution, len(patterns))
	for range patterns {
		r := <-ch
		collected[r.Index] = r
	}
	return collected
}

func renderMultiPatternOutput(collected []formattedPatternExecution, patterns []string, opts SearchOptions) string {
	var sb strings.Builder
	grouped := groupPatternSymbolBundles(collected)
	for i, pr := range collected {
		if sb.Len() > 0 {
			sb.WriteString("\n")
		}
		if wroteBundle := writeGroupedSymbolBundleOutput(&sb, grouped, pr, opts); wroteBundle {
			continue
		}
		fmt.Fprintf(&sb, "━━ Pattern %d/%d: %q ━━\n", i+1, len(patterns), pr.Pattern)
		sb.WriteString(pr.Output)
		if !strings.HasSuffix(pr.Output, "\n") {
			sb.WriteString("\n")
		}
	}

	if idx := buildCrossPatternIndexFromExecutions(collected, opts.LocatorRegistry, opts); idx != "" {
		sb.WriteString(idx)
	}
	return sb.String() + lineRangeHint
}

func writeGroupedSymbolBundleOutput(sb *strings.Builder, grouped map[string]patternSymbolBundleGroup, execution formattedPatternExecution, opts SearchOptions) bool {
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

func writeMultiPatternCache(cache tools.ToolCacheInterface, patterns []string, opts SearchOptions, output string, collected []formattedPatternExecution) {
	if cache == nil {
		return
	}
	multiKey := buildMultiCacheKey(patterns)
	cacheKey := buildMultiSearchCacheKey(opts, patterns)
	affectedFiles := collectAffectedFilesFromExecutions(collected, opts)
	cache.SetSearch(multiKey, cacheKey, output, affectedFiles)
}

type patternSymbolBundleGroup struct {
	Bundle   *SymbolBundle
	Patterns []string
	Emitted  bool
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
