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

type multiPatternCacheWrite struct {
	PatternKey    string
	CacheKey      string
	AffectedFiles []string
	Observation   *tools.RuntimeObservation
}

// executeMultiplePatternsDetailed は複数パターンを goroutine 並列で検索する。
// 各パターンは single pattern execution に委譲し、symbol fast path + キャッシュが自動で効く。
func executeMultiplePatternsDetailed(cache tools.ToolCacheInterface, contexts []singlePatternExecutionContext, opts SearchOptions) searchPatternsExecution {
	collected := collectMultiPatternExecutions(cache, contexts)
	output := renderMultiPatternOutput(collected, len(contexts), opts)
	writeMultiPatternCache(cache, contexts, opts, output, collected)
	return searchPatternsExecution{
		Output:              output,
		AffectedFiles:       collectAffectedFilesFromExecutions(collected, opts),
		Observation:         collectObservationFromExecutions(collected),
		PatternObservations: collectObservationByPatternFromExecutions(collected),
	}
}

func collectMultiPatternExecutions(cache tools.ToolCacheInterface, contexts []singlePatternExecutionContext) []formattedPatternExecution {
	ch := make(chan formattedPatternExecution, len(contexts))
	for i, ctx := range contexts {
		go runMultiPatternExecutionWorker(cache, i, ctx, ch)
	}
	return collectOrderedPatternExecutions(len(contexts), ch)
}

func runMultiPatternExecutionWorker(cache tools.ToolCacheInterface, idx int, context singlePatternExecutionContext, ch chan<- formattedPatternExecution) {
	result := executeSinglePatternWithContext(cache, context)
	ch <- buildFormattedPatternExecution(idx, result)
}

func buildFormattedPatternExecution(idx int, result singlePatternExecution) formattedPatternExecution {
	result.Output = strings.TrimSuffix(result.Output, lineRangeHint)
	return formattedPatternExecution{Index: idx, singlePatternExecution: result}
}

func collectOrderedPatternExecutions(count int, ch <-chan formattedPatternExecution) []formattedPatternExecution {
	collected := make([]formattedPatternExecution, count)
	for i := 0; i < count; i++ {
		r := <-ch
		collected[r.Index] = r
	}
	return collected
}

func writeMultiPatternCache(cache tools.ToolCacheInterface, contexts []singlePatternExecutionContext, opts SearchOptions, output string, collected []formattedPatternExecution) {
	if cache == nil {
		return
	}
	write := prepareMultiPatternCacheWrite(contexts, opts, collected)
	cache.SetSearch(write.PatternKey, write.CacheKey, output, write.AffectedFiles)
	storeMultiPatternObservation(write.PatternKey, write.CacheKey, write.Observation)
}

func prepareMultiPatternCacheWrite(contexts []singlePatternExecutionContext, opts SearchOptions, collected []formattedPatternExecution) multiPatternCacheWrite {
	return multiPatternCacheWrite{
		PatternKey:    buildMultiCacheKeyFromContexts(contexts),
		CacheKey:      buildMultiSearchCacheKeyFromContexts(opts, contexts),
		AffectedFiles: collectAffectedFilesFromExecutions(collected, opts),
		Observation:   collectObservationFromExecutions(collected),
	}
}

func collectObservationFromExecutions(collected []formattedPatternExecution) *tools.RuntimeObservation {
	observations := make([]*tools.RuntimeObservation, 0, len(collected))
	for _, execution := range collected {
		observations = append(observations, execution.Observation)
	}
	return tools.MergeRuntimeObservations(observations...)
}

func collectObservationByPatternFromExecutions(collected []formattedPatternExecution) map[string]*tools.RuntimeObservation {
	groups := make(map[string]*tools.RuntimeObservation)
	for _, execution := range collected {
		if observation := tools.CloneRuntimeObservation(execution.Observation); observation != nil {
			groups[execution.Pattern] = observation
		}
	}
	if len(groups) == 0 {
		return nil
	}
	return groups
}
