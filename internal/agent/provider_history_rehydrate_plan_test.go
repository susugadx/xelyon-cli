package agent

import (
	"reflect"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/ledger"
)

func TestProviderHistoryAppliedEvidencePointersFiltersDedupesAndCopies(t *testing.T) {
	readPointer := ledger.EvidencePointer{
		Path:       "README.md",
		StartLine:  1,
		EndLine:    20,
		Source:     "read_file",
		ToolCallID: "call_read",
		FileHash:   "first",
	}
	readDuplicate := readPointer
	readDuplicate.FileHash = "duplicate"
	searchPointer := ledger.EvidencePointer{
		Path:       "internal/search.go",
		StartLine:  10,
		EndLine:    12,
		Source:     "search_code",
		ToolCallID: "call_search",
	}
	gatherPointer := ledger.EvidencePointer{
		Path:       "internal/context.go",
		StartLine:  30,
		EndLine:    40,
		Source:     "gather_context",
		ToolCallID: "call_gather",
	}
	report := ProviderHistoryProjectionReport{
		Candidates: []ProviderHistoryReductionCandidate{
			{
				ToolName:           "read_file",
				ToolCallID:         "call_read",
				ReplacementApplied: true,
				EvidencePointers:   []ledger.EvidencePointer{readPointer, readDuplicate},
			},
			{
				ToolName:           "search_code",
				ToolCallID:         "call_unapplied",
				ReplacementApplied: false,
				EvidencePointers:   []ledger.EvidencePointer{searchPointer},
			},
			{
				ToolName:           "bash",
				ToolCallID:         "call_command",
				ReplacementApplied: true,
				EvidencePointers: []ledger.EvidencePointer{{
					Path:       "scripts/build.sh",
					StartLine:  1,
					EndLine:    2,
					Source:     "bash",
					ToolCallID: "call_command",
				}},
			},
			{
				ToolName:           "search_code",
				ToolCallID:         "call_search",
				ReplacementApplied: true,
				EvidencePointers:   []ledger.EvidencePointer{searchPointer},
			},
			{
				ToolName:           "gather_context",
				ToolCallID:         "call_gather",
				ReplacementApplied: true,
				EvidencePointers:   []ledger.EvidencePointer{gatherPointer},
			},
		},
		CommandEditDryRun: ProviderHistoryCommandEditDryRunReport{
			EditArgReplacedCount: 1,
			Candidates: []ProviderHistoryCommandEditDryRunCandidate{{
				ToolName:   "bash",
				ToolCallID: "call_command_dry_run",
			}, {
				ToolName:   "write_file",
				ToolCallID: "call_write",
				Reason:     "write_file_content",
			}},
		},
	}

	got := providerHistoryAppliedEvidencePointers(report)
	want := []ledger.EvidencePointer{readPointer, searchPointer, gatherPointer}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("providerHistoryAppliedEvidencePointers() = %#v, want %#v", got, want)
	}

	got[0].Path = "mutated.go"
	if report.Candidates[0].EvidencePointers[0].Path != "README.md" {
		t.Fatalf("providerHistoryAppliedEvidencePointers() did not return a defensive copy: %#v", report.Candidates[0].EvidencePointers)
	}
}

func TestProviderHistoryRehydratePlanEmptyWithoutLedgerOrAppliedEvidence(t *testing.T) {
	store := ledger.NewStoreWithRoot(t.TempDir())
	tests := []struct {
		name  string
		agent *Agent
	}{
		{name: "nil agent"},
		{name: "nil runtime", agent: &Agent{}},
		{name: "nil task ledger", agent: &Agent{Runtime: &AgentRuntime{}}},
		{name: "empty report", agent: &Agent{Runtime: &AgentRuntime{TaskLedger: store}}},
		{name: "unapplied candidate", agent: &Agent{Runtime: &AgentRuntime{
			TaskLedger: store,
			LastProviderHistoryProjectionReport: ProviderHistoryProjectionReport{Candidates: []ProviderHistoryReductionCandidate{{
				ToolName:         "read_file",
				ToolCallID:       "call_read",
				EvidencePointers: []ledger.EvidencePointer{{Path: "src/main.go", StartLine: 1, EndLine: 2, Source: "read_file"}},
			}}},
		}}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if plan := tt.agent.buildProviderHistoryRehydratePlan([]string{"src/main.go"}); len(plan.Items) != 0 {
				t.Fatalf("buildProviderHistoryRehydratePlan() = %#v, want empty plan", plan)
			}
		})
	}
}

func TestProviderHistoryRehydratePlanBuildsFromAppliedReportAndEditReadinessObservation(t *testing.T) {
	taskLedger := providerHistoryTaskLedgerWithEvidence(t,
		providerHistoryEvidenceItem{
			ToolName:   "read_file",
			ToolCallID: "call_old_read",
			Path:       "src/main.go",
			StartLine:  10,
			EndLine:    20,
		},
	)
	taskLedger.RecordEditReadinessObservation(ledger.EditReadinessObservation{
		Path:           "src/main.go",
		NormalizedPath: "src/main.go",
		Status:         ledger.EditReadinessStatusWarning,
		Reasons:        []ledger.EditReadinessReason{ledger.EditReadinessReasonNoRecentRead},
	})
	agent := &Agent{
		Runtime: &AgentRuntime{TaskLedger: taskLedger},
		History: []api.Message{
			providerHistoryAssistantToolCall("call_old_read", "read_file"),
			providerHistoryToolResult("call_old_read", "read_file", strings.Repeat("old read_file output\n", 40)),
			{Role: "assistant", Content: "after old read"},
			providerHistoryAssistantToolCall("call_latest", "read_file"),
			providerHistoryToolResult("call_latest", "read_file", "latest read_file output"),
			{Role: "assistant", Content: "done"},
		},
	}

	result := agent.buildProviderHistoryProjection(ProviderHistoryReductionPolicy{Mode: ProviderHistoryReductionApply})
	candidate := candidateByToolCallID(result.Report, "call_old_read")
	if candidate == nil || !candidate.ReplacementApplied || len(candidate.EvidencePointers) != 1 {
		t.Fatalf("apply report candidate = %#v, want applied read_file candidate with evidence", candidate)
	}
	agent.recordLastProviderHistoryProjectionReport(result.Report)

	plan := agent.buildProviderHistoryRehydratePlan(nil)
	want := ledger.RehydratePlan{Items: []ledger.RehydratePlanItem{{
		Path:       "src/main.go",
		StartLine:  10,
		EndLine:    20,
		Source:     "read_file",
		Reason:     ledger.RehydratePlanReasonEditTargetMissingEvidence,
		ToolCallID: "call_old_read",
	}}}
	if !reflect.DeepEqual(plan, want) {
		t.Fatalf("buildProviderHistoryRehydratePlan() = %#v, want %#v", plan, want)
	}
}
