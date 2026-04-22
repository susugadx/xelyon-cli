package agent

import "context"

func (a *Agent) executeReadFileBatchMerge(ctx context.Context, state *parallelToolCallState) {
	execFlags := make([]bool, len(state.allToolCalls))
	for i, entry := range state.entries {
		execFlags[i] = entry.status == parallelToolCallStatusExecute
	}
	readBatchSegments := segmentReadFileBatches(state.allToolCalls, execFlags)

	for _, seg := range readBatchSegments {
		for callStart, pathStart := 0, 0; callStart < len(seg.indices); {
			chunkCallStart := callStart
			chunkPathStart := pathStart
			chunkPathCount := 0

			for callStart < len(seg.indices) {
				callPathCount := seg.pathCounts[callStart]
				if chunkPathCount > 0 && chunkPathCount+callPathCount > maxReadFileBatchPaths {
					break
				}
				chunkPathCount += callPathCount
				pathStart += callPathCount
				callStart++
			}

			chunkIndices := seg.indices[chunkCallStart:callStart]
			chunkPathCounts := seg.pathCounts[chunkCallStart:callStart]
			chunkPaths := seg.paths[chunkPathStart : chunkPathStart+chunkPathCount]

			if len(chunkIndices) < 2 {
				continue
			}

			mergedResult := a.executeReadFileBatch(ctx, chunkPaths)
			perFile := splitReadFileBatchResult(mergedResult, chunkPaths)
			if perFile == nil {
				continue
			}

			offset := 0
			for j, idx := range chunkIndices {
				callPaths := chunkPaths[offset : offset+chunkPathCounts[j]]
				offset += chunkPathCounts[j]
				if section, ok := joinReadFileBatchSections(perFile, callPaths); ok {
					state.results[idx] = toolExecResult{result: section}
				} else {
					state.results[idx] = toolExecResult{result: mergedResult}
				}
				state.entries[idx] = parallelToolCallEntry{status: parallelToolCallStatusBatched}
			}
			a.recordReadFileBatchMerge(len(chunkIndices))
		}
	}
}
