package agent

import (
	"context"

	"github.com/susugadx/xelyon-cli/internal/toolruntime"
	"github.com/susugadx/xelyon-cli/internal/tools"
	filetool "github.com/susugadx/xelyon-cli/internal/tools/file"
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
			mergedResult := mergedBatchResult.Execution.Result
			if len(mergedBatchResult.Sections) != len(chunkPaths) {
				continue
			}

			offset := 0
			for j, idx := range chunkIndices {
				callPaths := chunkPaths[offset : offset+chunkPathCounts[j]]
				callSections := readFileBatchCallSections(mergedBatchResult.Sections, offset, chunkPathCounts[j])
				offset += chunkPathCounts[j]
				if len(callSections) == len(callPaths) {
					section := filetool.RenderReadExecutionSections(callSections)
					state.Results[idx] = toolruntime.Result{
						Result:      section,
						Observation: filetool.MergeReadExecutionSectionObservations(callSections),
						Error:       tools.IsErrorResult(section),
					}
				} else {
					state.Results[idx] = toolruntime.Result{
						Result:      mergedResult,
						Observation: mergedBatchResult.Execution.Observation,
						Error:       mergedBatchResult.Execution.Error,
					}
				}
				state.Entries[idx] = toolruntime.ParallelCallEntry{Status: toolruntime.ParallelCallStatusBatched}
			}
			a.recordReadFileBatchMerge(len(chunkIndices))
		}
	}
}

func readFileBatchCallSections(sections []filetool.ReadExecutionSection, offset, count int) []filetool.ReadExecutionSection {
	if count <= 0 || offset < 0 || offset+count > len(sections) {
		return nil
	}
	return sections[offset : offset+count]
}
