package agent

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

type planStepToolResultHandler struct {
	runner  *TurnRunner
	tracker *MutationTracker
	state   *stepRunState
}

func newPlanStepToolResultHandler(r *TurnRunner, state *stepRunState) *planStepToolResultHandler {
	return &planStepToolResultHandler{
		runner:  r,
		tracker: r.mutationTracker(),
		state:   state,
	}
}

func (h *planStepToolResultHandler) Handle(toolCall *tools.ToolCall, result string, change *tools.FileChange) {
	a := h.runner.agent
	a.appendSessionToolExecution(toolCall, result)
	h.tracker.RecordToolResult(toolCall, result, change)
	h.trackWriteState(toolCall, result)
	h.trackFailure(toolCall, result)
	a.appendToolResultToHistory(toolCall, result)
}

func (h *planStepToolResultHandler) trackWriteState(toolCall *tools.ToolCall, result string) {
	if !tools.IsWriteTool(toolCall.Tool) {
		return
	}
	h.state.stepHadWrites = true
	if strings.Contains(result, "no files found") ||
		strings.Contains(result, "Total matches: 0") ||
		strings.Contains(result, "no change needed") {
		h.state.stepHadNoChangeNeeded = true
	}
}

func (h *planStepToolResultHandler) trackFailure(toolCall *tools.ToolCall, result string) {
	if toolCall.Tool != "bash" && !tools.IsWriteTool(toolCall.Tool) {
		return
	}
	if failed, reason := plan.ContainsFailure(result); failed {
		h.state.lastFailedResult = result
		h.state.lastFailReason = reason
	}
}
