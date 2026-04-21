package search

import (
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

func writeMultiPatternCache(cache tools.ToolCacheInterface, patterns []string, opts SearchOptions, output string, collected []formattedPatternExecution) {
	if cache == nil {
		return
	}
	multiKey := buildMultiCacheKey(patterns)
	cacheKey := buildMultiSearchCacheKey(opts, patterns)
	affectedFiles := collectAffectedFilesFromExecutions(collected, opts)
	cache.SetSearch(multiKey, cacheKey, output, affectedFiles)
}
