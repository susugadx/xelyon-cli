package agent

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/taskstate"
)

const (
	providerHistoryRehydrateTestToolName   = "read_file"
	providerHistoryRehydrateTestToolCallID = "call_rehydrate"
)

func recordProviderHistoryRehydratePlanFixture(store *taskstate.Store, path string, startLine, endLine int) ProviderHistoryProjectionReport {
	if store != nil {
		store.RecordEditReadinessObservation(taskstate.EditReadinessObservation{
			Path:           path,
			NormalizedPath: path,
			Status:         taskstate.EditReadinessStatusWarning,
			Reasons:        []taskstate.EditReadinessReason{taskstate.EditReadinessReasonNoRecentRead},
		})
	}
	return ProviderHistoryProjectionReport{
		Mode:           ProviderHistoryReductionApply,
		CandidateCount: 1,
		ReplacedCount:  1,
		Candidates: []ProviderHistoryReductionCandidate{
			providerHistoryRehydratePlanTestCandidate(path, startLine, endLine),
		},
	}
}

func installProviderHistoryRehydratePlanFixture(t *testing.T, agent *Agent, path string, startLine, endLine int) {
	t.Helper()
	store := taskstate.NewStoreWithRoot(t.TempDir())
	agent.Runtime.TaskLedger = store
	agent.Runtime.LastProviderHistoryProjectionReport = recordProviderHistoryRehydratePlanFixture(store, path, startLine, endLine)
}

func providerHistoryRehydratePlanTestCandidate(path string, startLine, endLine int) ProviderHistoryReductionCandidate {
	return ProviderHistoryReductionCandidate{
		ToolName:           providerHistoryRehydrateTestToolName,
		ToolCallID:         providerHistoryRehydrateTestToolCallID,
		ReplacementApplied: true,
		EvidencePointers: []taskstate.EvidencePointer{{
			Path:       path,
			StartLine:  startLine,
			EndLine:    endLine,
			Source:     providerHistoryRehydrateTestToolName,
			ToolCallID: providerHistoryRehydrateTestToolCallID,
			PathBase:   taskstate.EvidencePointerPathBaseRepoRoot,
		}},
	}
}
