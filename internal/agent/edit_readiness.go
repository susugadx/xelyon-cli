package agent

import (
	"context"

	"github.com/susugadx/xelyon-cli/internal/taskstate"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

func (a *Agent) observeEditReadinessBeforeTool(ctx context.Context, toolCall *tools.ToolCall) {
	store := a.editReadinessStore()
	if store == nil {
		return
	}
	extraction := extractEditReadinessTargets(toolCall)
	if len(extraction.targets) == 0 && !extraction.unknown {
		return
	}
	if extraction.unknown {
		store.RecordEditReadinessObservation(taskstate.EditReadinessObservation{
			ToolName:   editReadinessToolName(toolCall),
			ToolCallID: editReadinessToolCallID(toolCall),
			Status:     taskstate.EditReadinessStatusUnknown,
		})
		return
	}
	for _, target := range extraction.targets {
		observation := store.CheckEditReadiness(ctx, target, taskstate.EditReadinessOptions{})
		store.RecordEditReadinessObservation(observation)
	}
}

func (a *Agent) editReadinessStore() *taskstate.Store {
	if a == nil || a.Runtime == nil {
		return nil
	}
	return a.Runtime.TaskLedger
}
