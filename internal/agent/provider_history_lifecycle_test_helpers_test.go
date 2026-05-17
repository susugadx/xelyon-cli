package agent

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

const providerHistoryStaleLedgerCallID = "call_rescue_001"

type providerHistoryStaleLedgerFixture struct {
	CallID  string
	OldRead string
	History []api.Message
}

func newProviderHistoryStaleLedgerFixture() providerHistoryStaleLedgerFixture {
	oldRead := providerHistoryStaleLedgerReadOutput()
	return providerHistoryStaleLedgerFixture{
		CallID:  providerHistoryStaleLedgerCallID,
		OldRead: oldRead,
		History: providerHistoryReductionRequestHistory(providerHistoryStaleLedgerCallID, oldRead),
	}
}

func providerHistoryStaleLedgerReadOutput() string {
	return strings.Repeat("old provider-facing read_file output with reused call id\n", 5)
}

func seedProviderHistoryReductionStaleLedgerEvidence(t *testing.T, agent *Agent, fixture providerHistoryStaleLedgerFixture) {
	t.Helper()
	agent.Runtime.Options.EnableProviderHistoryReduction = true
	agent.Runtime.TaskLedger = providerHistoryTaskLedgerWithEvidence(t,
		providerHistoryEvidenceItem{ToolName: "read_file", ToolCallID: fixture.CallID, Path: "stale.go", StartLine: 10},
	)
}

func assertProviderHistoryReductionDoesNotUseStaleLedgerEvidence(t *testing.T, agent *Agent, fixture providerHistoryStaleLedgerFixture) {
	t.Helper()

	projection := agent.providerFacingHistory()

	if len(projection) < 2 {
		t.Fatalf("provider projection length = %d, want reused call fixture history", len(projection))
	}
	if projection[1].Content != fixture.OldRead {
		t.Fatalf("provider old tool result = %q, want raw candidate after stale ledger reset", projection[1].Content)
	}
	candidate := candidateByToolCallID(agent.Runtime.LastProviderHistoryProjectionReport, fixture.CallID)
	if candidate == nil || candidate.ReplacementApplied || candidate.KeepReason != "missing_evidence_pointer" {
		t.Fatalf("candidate = %#v, want missing_evidence_pointer without stale replacement", candidate)
	}
}
