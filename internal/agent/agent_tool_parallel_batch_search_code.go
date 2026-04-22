package agent

import (
	"context"
	"strings"
)

func (a *Agent) executeSearchCodeBatch(ctx context.Context, state *parallelToolCallState) {
	type searchBatchGroup struct {
		indices  []int
		patterns []string
	}

	searchGroups := make(map[string]*searchBatchGroup)
	var searchGroupOrder []string

	for i, entry := range state.entries {
		if entry.status != parallelToolCallStatusExecute {
			continue
		}
		tc := state.allToolCalls[i]
		if tc.Tool != "search_code" || !isSimpleSearchPattern(tc.Args["pattern"]) {
			continue
		}
		optKey := searchCodeOptionsKey(tc)
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
		for chunkStart := 0; chunkStart < len(group.indices); chunkStart += maxSearchBatchPatterns {
			chunkEnd := chunkStart + maxSearchBatchPatterns
			if chunkEnd > len(group.indices) {
				chunkEnd = len(group.indices)
			}
			chunkIndices := group.indices[chunkStart:chunkEnd]
			chunkPatterns := group.patterns[chunkStart:chunkEnd]

			if len(chunkIndices) < 2 {
				continue
			}

			leaderTC := state.allToolCalls[chunkIndices[0]]
			mergedTC := cloneToolCallWithNewPattern(leaderTC, strings.Join(chunkPatterns, ","))
			mergedResult, _ := a.executeToolForParallel(ctx, mergedTC)

			perPattern := splitMultiPatternResult(mergedResult, chunkPatterns)
			if perPattern == nil {
				continue
			}

			for j, idx := range chunkIndices {
				pattern := chunkPatterns[j]
				if section, ok := perPattern[pattern]; ok {
					state.results[idx] = toolExecResult{result: section}
				} else {
					state.results[idx] = toolExecResult{result: mergedResult}
				}
				state.entries[idx] = parallelToolCallEntry{status: parallelToolCallStatusBatched}
			}
			a.recordSearchCodeBatchMerge(len(chunkIndices))
		}
	}
}
