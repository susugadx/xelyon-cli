package agent

import (
	"context"

	"github.com/susugadx/xelyon-cli/internal/toolruntime"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

func (a *Agent) executeReadFileBatchMerge(ctx context.Context, state *toolruntime.ParallelCallState) {
	execFlags := make([]bool, len(state.AllToolCalls))
	for i, entry := range state.Entries {
		execFlags[i] = entry.Status == toolruntime.ParallelCallStatusExecute
	}
	readBatchSegments := toolruntime.SegmentReadFileBatches(state.AllToolCalls, execFlags)

	for _, seg := range readBatchSegments {
		for callStart, pathStart := 0, 0; callStart < len(seg.Indices); {
			chunkCallStart := callStart
			chunkPathStart := pathStart
			nextCallStart, chunkPathCount := nextPathBudgetChunk(seg.PathCounts, callStart, toolruntime.MaxReadFileBatchPaths)
			callStart = nextCallStart
			pathStart += chunkPathCount

			chunkIndices := seg.Indices[chunkCallStart:callStart]
			chunkPathCounts := seg.PathCounts[chunkCallStart:callStart]
			chunkPaths := seg.Paths[chunkPathStart : chunkPathStart+chunkPathCount]

			if len(chunkIndices) < 2 {
				continue
			}

			for _, idx := range chunkIndices {
				a.emitTUIToolRunning(state.AllToolCalls[idx])
			}
			mergedBatchResult := a.executeReadFileBatchResult(ctx, chunkPaths)
			mergedResult := mergedBatchResult.Result
			perFile := toolruntime.SplitReadFileBatchResult(mergedResult, chunkPaths)
			if perFile == nil {
				continue
			}

			offset := 0
			for j, idx := range chunkIndices {
				callPaths := chunkPaths[offset : offset+chunkPathCounts[j]]
				offset += chunkPathCounts[j]
				if section, ok := toolruntime.JoinReadFileBatchSections(perFile, callPaths); ok {
					state.Results[idx] = toolruntime.Result{
						Result: section,
						Error:  tools.IsErrorResult(section),
					}
				} else {
					state.Results[idx] = toolruntime.Result{
						Result: mergedResult,
						Error:  mergedBatchResult.Error,
					}
				}
				state.Entries[idx] = toolruntime.ParallelCallEntry{Status: toolruntime.ParallelCallStatusBatched}
			}
			a.recordReadFileBatchMerge(len(chunkIndices))
		}
	}
}
