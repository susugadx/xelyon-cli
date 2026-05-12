package agent

import (
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
	"github.com/susugadx/xelyon-cli/internal/toolruntime"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

type normalModeToolResultHandler struct {
	runner           *TurnRunner
	tracker          *MutationTracker
	turnMutations    *turnMutationState
	lastFailedResult string
}

func newNormalModeToolResultHandler(r *TurnRunner, state *normalModeState) *normalModeToolResultHandler {
	var turnMutations *turnMutationState
	if state != nil {
		turnMutations = &state.turnMutations
	}
	return &normalModeToolResultHandler{
		runner:        r,
		tracker:       r.mutationTracker(),
		turnMutations: turnMutations,
	}
}

func (h *normalModeToolResultHandler) Handle(tc *tools.ToolCall, execResult toolruntime.Result) {
	a := h.runner.agent
	result := execResult.Result
	a.appendSessionToolExecution(tc, result, execResult.Error)

	if a.handleStrReplaceErrors(tc, result) {
		return
	}
	if a.handleCommentFlow(tc, result) {
		return
	}

	h.tracker.RecordToolExecutionResult(tc, execResult, h.turnMutations)
	a.appendToolResultToHistory(tc, result)
	_, _ = fmt.Fprintln(a.output())

	if tc.Tool == "bash" || tools.IsWriteTool(tc.Tool) {
		if failed, _ := plan.ContainsFailure(result); failed {
			h.lastFailedResult = result
		}
	}
}

func (h *normalModeToolResultHandler) LastFailedResult() string {
	if h == nil {
		return ""
	}
	return h.lastFailedResult
}
