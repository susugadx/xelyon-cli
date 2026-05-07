package agent

import (
	"context"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/toolruntime"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

func (a *Agent) executeSearchCodeBatch(ctx context.Context, state *toolruntime.ParallelCallState) {
	type searchBatchGroup struct {
		indices  []int
		patterns []string
	}

	searchGroups := make(map[string]*searchBatchGroup)
	var searchGroupOrder []string

	for i, entry := range state.Entries {
		if entry.Status != toolruntime.ParallelCallStatusExecute {
			continue
		}
		tc := state.AllToolCalls[i]
		if tc.Tool != "search_code" || !toolruntime.IsSimpleSearchPattern(tc.Args["pattern"]) {
			continue
		}
		optKey := toolruntime.SearchCodeOptionsKey(tc)
		if group, ok := searchGroups[optKey]; ok {
			group.indices = append(group.indices, i)
			group.patterns = append(group.patterns, tc.Args["pattern"])
		} else {
			searchGroups[optKey] = &searchBatchGroup{
				indices:  []int{i},
				patterns: []string{tc.Args["pattern"]},
			}
			searchGroupOrder = append(searchGroupOrder, optKey)
		}
	}

	for _, optKey := range searchGroupOrder {
		group := searchGroups[optKey]
		if len(group.indices) < 2 {
			continue
		}
		forEachIndexChunk(len(group.indices), toolruntime.MaxSearchBatchPatterns, func(chunkStart, chunkEnd int) {
			chunkIndices := group.indices[chunkStart:chunkEnd]
			chunkPatterns := group.patterns[chunkStart:chunkEnd]

			if len(chunkIndices) < 2 {
				return
			}

			for _, idx := range chunkIndices {
				a.emitTUIToolRunning(state.AllToolCalls[idx])
			}
			leaderTC := state.AllToolCalls[chunkIndices[0]]
			mergedTC := toolruntime.CloneToolCallWithNewPattern(leaderTC, strings.Join(chunkPatterns, ","))
			mergedExecResult := a.executeToolForParallelResult(ctx, mergedTC)
			mergedResult := mergedExecResult.Result

			perPattern := toolruntime.SplitMultiPatternResult(mergedResult, chunkPatterns)
			if perPattern == nil {
				return
			}

			for j, idx := range chunkIndices {
				pattern := chunkPatterns[j]
				if section, ok := perPattern[pattern]; ok {
					state.Results[idx] = toolruntime.Result{
						Result: section,
						Error:  tools.IsErrorResult(section),
					}
				} else {
					state.Results[idx] = toolruntime.Result{
						Result: mergedResult,
						Error:  mergedExecResult.Error,
					}
				}
				state.Entries[idx] = toolruntime.ParallelCallEntry{Status: toolruntime.ParallelCallStatusBatched}
			}
			a.recordSearchCodeBatchMerge(len(chunkIndices))
		})
	}
}
