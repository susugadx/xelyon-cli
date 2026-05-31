package agent

import (
	"context"

	"github.com/susugadx/xelyon-cli/internal/taskstate"
)

// RehydrateEvidencePointer は agent runtime の ledger で evidence pointer を再読込する。
func (a *Agent) RehydrateEvidencePointer(ctx context.Context, pointer taskstate.EvidencePointer) (taskstate.EvidenceRehydrateResult, error) {
	if a == nil || a.Runtime == nil || a.Runtime.TaskLedger == nil {
		result := taskstate.EvidenceRehydrateResult{
			Path:      pointer.Path,
			StartLine: pointer.StartLine,
			EndLine:   pointer.EndLine,
			Reason:    taskstate.EvidenceRehydrateReasonWorkspaceUnavailable,
		}
		return result, &taskstate.EvidenceRehydrateError{
			Reason: taskstate.EvidenceRehydrateReasonWorkspaceUnavailable,
			Path:   pointer.Path,
		}
	}
	return a.Runtime.TaskLedger.RehydrateEvidencePointer(ctx, pointer)
}
