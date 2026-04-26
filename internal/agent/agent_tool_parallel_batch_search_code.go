package agent

import (
	"context"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/toolruntime"
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
		for chunkStart := 0; chunkStart < len(group.indices); chunkStart += toolruntime.MaxSearchBatchPatterns {
			chunkEnd := chunkStart + toolruntime.MaxSearchBatchPatterns
			if chunkEnd > len(group.indices) {
				chunkEnd = len(group.indices)
			}
			chunkIndices := group.indices[chunkStart:chunkEnd]
			chunkPatterns := group.patterns[chunkStart:chunkEnd]

			if len(chunkIndices) < 2 {
				continue
			}

			leaderTC := state.AllToolCalls[chunkIndices[0]]
			mergedTC := toolruntime.CloneToolCallWithNewPattern(leaderTC, strings.Join(chunkPatterns, ","))
			mergedResult, _ := a.executeToolForParallel(ctx, mergedTC)

			perPattern := toolruntime.SplitMultiPatternResult(mergedResult, chunkPatterns)
			if perPattern == nil {
				continue
			}

			for j, idx := range chunkIndices {
				pattern := chunkPatterns[j]
				if section, ok := perPattern[pattern]; ok {
					state.Results[idx] = toolruntime.Result{Result: section}
				} else {
					state.Results[idx] = toolruntime.Result{Result: mergedResult}
				}
				state.Entries[idx] = toolruntime.ParallelCallEntry{Status: toolruntime.ParallelCallStatusBatched}
			}
			a.recordSearchCodeBatchMerge(len(chunkIndices))
		}
	}
}
