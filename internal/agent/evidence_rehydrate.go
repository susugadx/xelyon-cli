package agent

import (
	"context"

	"github.com/susugadx/xelyon-cli/internal/ledger"
)

// RehydrateEvidencePointer は agent runtime の ledger で evidence pointer を再読込する。
func (a *Agent) RehydrateEvidencePointer(ctx context.Context, pointer ledger.EvidencePointer) (ledger.EvidenceRehydrateResult, error) {
	if a == nil || a.Runtime == nil || a.Runtime.TaskLedger == nil {
		result := ledger.EvidenceRehydrateResult{
			Path:      pointer.Path,
			StartLine: pointer.StartLine,
			EndLine:   pointer.EndLine,
			Reason:    ledger.EvidenceRehydrateReasonWorkspaceUnavailable,
		}
		return result, &ledger.EvidenceRehydrateError{
			Reason: ledger.EvidenceRehydrateReasonWorkspaceUnavailable,
			Path:   pointer.Path,
		}
	}
	return a.Runtime.TaskLedger.RehydrateEvidencePointer(ctx, pointer)
}
