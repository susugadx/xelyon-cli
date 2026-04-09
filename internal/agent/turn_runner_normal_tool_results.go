package agent

import (
	"fmt"

	"github.com/susugadx/xelyon-cli/internal/agent/plan"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

type normalModeToolResultHandler struct {
	runner           *TurnRunner
	tracker          *MutationTracker
	lastFailedResult string
}

func newNormalModeToolResultHandler(r *TurnRunner) *normalModeToolResultHandler {
	return &normalModeToolResultHandler{
		runner:  r,
		tracker: r.mutationTracker(),
	}
}

func (h *normalModeToolResultHandler) Handle(tc *tools.ToolCall, result string, change *tools.FileChange) {
	a := h.runner.agent
	a.appendSessionToolExecution(tc, result)

	if a.handleStrReplaceErrors(tc, result) {
		return
	}
	if a.handleCommentFlow(tc, result) {
		return
	}

	h.tracker.RecordToolResult(tc, result, change)
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
