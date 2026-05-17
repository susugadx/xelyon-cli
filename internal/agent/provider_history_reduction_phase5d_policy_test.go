package agent

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

func TestPhase5DComplexMixedHistoryReducesOnlyOldReadSearchGather(t *testing.T) {
	outputs := map[string]string{
		"call_read_old":   phase5DOutput("old read_file mixed output"),
		"call_search_old": phase5DOutput("old search_code mixed output"),
		"call_gather_old": phase5DOutput("old gather_context mixed output"),
		"call_patch":      phase5DOutput("apply_patch output"),
		"call_str":        phase5DOutput("str_replace output"),
		"call_bash":       phase5DOutput("bash output"),
		"call_write":      phase5DOutput("write_file output"),
		"call_delete":     phase5DOutput("delete_file output"),
	}
	taskLedger := providerHistoryTaskLedgerWithEvidence(t,
		providerHistoryEvidenceItem{ToolName: "read_file", ToolCallID: "call_read_old", Path: "README.md", StartLine: 1},
		providerHistoryEvidenceItem{ToolName: "search_code", ToolCallID: "call_search_old", Path: "internal/search.go", StartLine: 10},
		providerHistoryEvidenceItem{ToolName: "gather_context", ToolCallID: "call_gather_old", Path: "internal/gather.go", StartLine: 20},
	)
	agent := &Agent{
		Runtime: &AgentRuntime{TaskLedger: taskLedger},
		History: []api.Message{
			{Role: "user", Content: "inspect mixed history"},
			providerHistoryAssistantToolCall("call_read_old", "read_file"),
			providerHistoryToolResult("call_read_old", "read_file", outputs["call_read_old"]),
			{Role: "assistant", Content: "read done"},
			providerHistoryAssistantToolCall("call_search_old", "search_code"),
			providerHistoryToolResult("call_search_old", "search_code", outputs["call_search_old"]),
			{Role: "assistant", Content: "search done"},
			providerHistoryAssistantToolCall("call_gather_old", "gather_context"),
			providerHistoryToolResult("call_gather_old", "gather_context", outputs["call_gather_old"]),
			{Role: "assistant", Content: "gather done"},
			providerHistoryAssistantToolCall("call_patch", "apply_patch"),
			providerHistoryToolResult("call_patch", "apply_patch", outputs["call_patch"]),
			{Role: "assistant", Content: "patch done"},
			providerHistoryAssistantToolCall("call_str", "str_replace"),
			providerHistoryToolResult("call_str", "str_replace", outputs["call_str"]),
			{Role: "assistant", Content: "replace done"},
			providerHistoryAssistantToolCall("call_bash", "bash"),
			providerHistoryToolResult("call_bash", "bash", outputs["call_bash"]),
			{Role: "assistant", Content: "bash done"},
			providerHistoryAssistantToolCall("call_write", "write_file"),
			providerHistoryToolResult("call_write", "write_file", outputs["call_write"]),
			{Role: "assistant", Content: "write done"},
			providerHistoryAssistantToolCall("call_delete", "delete_file"),
			providerHistoryToolResult("call_delete", "delete_file", outputs["call_delete"]),
			{Role: "assistant", Content: "delete done"},
			providerHistoryAssistantToolCalls(
				providerHistoryToolCall("call_tail_read", "read_file"),
				providerHistoryToolCall("call_tail_search", "search_code"),
			),
			providerHistoryToolResult("call_tail_read", "read_file", "latest trailing read output"),
			providerHistoryToolResult("call_tail_search", "search_code", "latest trailing search output"),
		},
	}

	result := agent.buildProviderHistoryProjection(ProviderHistoryReductionPolicy{Mode: ProviderHistoryReductionApply})

	for _, id := range []string{"call_read_old", "call_search_old", "call_gather_old"} {
		msg := phase5DToolResultByID(t, result.History, id)
		if !strings.HasPrefix(msg.Content, providerHistoryReductionPlaceholderPrefix) {
			t.Fatalf("%s projected content = %q, want placeholder", id, msg.Content)
		}
	}
	for _, id := range []string{"call_patch", "call_str", "call_bash", "call_write", "call_delete", "call_tail_read", "call_tail_search"} {
		msg := phase5DToolResultByID(t, result.History, id)
		if msg.Content != phase5DToolRawContent(id, outputs) {
			t.Fatalf("%s projected content = %q, want raw non-candidate/trailing content", id, msg.Content)
		}
		if strings.HasPrefix(msg.Content, providerHistoryReductionPlaceholderPrefix) {
			t.Fatalf("%s should never be reduced, got %q", id, msg.Content)
		}
	}
	report := result.Report
	if report.CandidateCount != 3 || report.ReplacedCount != 3 || report.ToolResultCount != 10 || report.KeptCount != 7 {
		t.Fatalf("report counts = candidates %d replaced %d toolResults %d kept %d, want 3/3/10/7", report.CandidateCount, report.ReplacedCount, report.ToolResultCount, report.KeptCount)
	}
	if report.EstimatedSavedBytes <= 0 {
		t.Fatalf("EstimatedSavedBytes = %d, want positive savings", report.EstimatedSavedBytes)
	}
	for _, id := range []string{"call_patch", "call_str", "call_bash", "call_write", "call_delete"} {
		assertKeepReason(t, report, id, "write_or_command_tool")
	}
	if got := countKeptByToolCallIDAndReason(report, "call_tail_read", "trailing_tool_suffix"); got != 1 {
		t.Fatalf("call_tail_read trailing keep count = %d, want 1", got)
	}
	if got := countKeptByToolCallIDAndReason(report, "call_tail_search", "trailing_tool_suffix"); got != 1 {
		t.Fatalf("call_tail_search trailing keep count = %d, want 1", got)
	}
	assertProviderHistoryByteMetrics(t, agent.History, result.History, report)
}

func TestPhase5DEvidencePointerSafetyRegressions(t *testing.T) {
	t.Run("missing evidence", func(t *testing.T) {
		agent := &Agent{History: phase5DReplaceableHistory("call_missing", "read_file", phase5DOutput("missing evidence"))}

		result := agent.buildProviderHistoryProjection(ProviderHistoryReductionPolicy{Mode: ProviderHistoryReductionApply})

		if result.Report.CandidateCount != 1 || result.Report.ReplacedCount != 0 {
			t.Fatalf("report counts = %d/%d, want one kept candidate", result.Report.CandidateCount, result.Report.ReplacedCount)
		}
		assertKeepReason(t, result.Report, "call_missing", "missing_evidence_pointer")
	})

	t.Run("ambiguous reused tool call id", func(t *testing.T) {
		agent := &Agent{
			Runtime: &AgentRuntime{TaskLedger: providerHistoryTaskLedgerWithEvidence(t,
				providerHistoryEvidenceItem{ToolName: "read_file", ToolCallID: "call_repeat", Path: "repeat.go", StartLine: 1},
			)},
			History: []api.Message{
				providerHistoryAssistantToolCall("call_repeat", "read_file"),
				providerHistoryToolResult("call_repeat", "read_file", phase5DOutput("first repeated read")),
				{Role: "assistant", Content: "first done"},
				providerHistoryAssistantToolCall("call_repeat", "read_file"),
				providerHistoryToolResult("call_repeat", "read_file", phase5DOutput("second repeated read")),
				{Role: "assistant", Content: "second done"},
				providerHistoryAssistantToolCall("call_latest", "read_file"),
				providerHistoryToolResult("call_latest", "read_file", "latest raw"),
				{Role: "assistant", Content: "done"},
			},
		}

		result := agent.buildProviderHistoryProjection(ProviderHistoryReductionPolicy{Mode: ProviderHistoryReductionApply})

		if result.Report.CandidateCount != 2 || result.Report.ReplacedCount != 0 {
			t.Fatalf("report counts = %d/%d, want 2/0", result.Report.CandidateCount, result.Report.ReplacedCount)
		}
		if got := countKeptByToolCallIDAndReason(result.Report, "call_repeat", "ambiguous_evidence_pointer"); got != 2 {
			t.Fatalf("ambiguous_evidence_pointer keep count = %d, want 2", got)
		}
	})

	t.Run("candidate shares evidence key with trailing kept result", func(t *testing.T) {
		agent := &Agent{
			Runtime: &AgentRuntime{TaskLedger: providerHistoryTaskLedgerWithEvidence(t,
				providerHistoryEvidenceItem{ToolName: "read_file", ToolCallID: "call_shared", Path: "shared.go", StartLine: 1},
			)},
			History: []api.Message{
				providerHistoryAssistantToolCall("call_shared", "read_file"),
				providerHistoryToolResult("call_shared", "read_file", phase5DOutput("old shared read")),
				{Role: "assistant", Content: "old done"},
				providerHistoryAssistantToolCall("call_shared", "read_file"),
				providerHistoryToolResult("call_shared", "read_file", "trailing shared read"),
			},
		}

		result := agent.buildProviderHistoryProjection(ProviderHistoryReductionPolicy{Mode: ProviderHistoryReductionApply})

		if result.Report.CandidateCount != 1 || result.Report.ReplacedCount != 0 {
			t.Fatalf("report counts = %d/%d, want 1/0", result.Report.CandidateCount, result.Report.ReplacedCount)
		}
		assertKeepReason(t, result.Report, "call_shared", "ambiguous_evidence_pointer")
		if got := countKeptByToolCallIDAndReason(result.Report, "call_shared", "trailing_tool_suffix"); got != 1 {
			t.Fatalf("trailing keep count = %d, want 1", got)
		}
	})

	t.Run("placeholder larger than raw content", func(t *testing.T) {
		agent := &Agent{
			Runtime: &AgentRuntime{TaskLedger: providerHistoryTaskLedgerWithEvidence(t,
				providerHistoryEvidenceItem{ToolName: "read_file", ToolCallID: "call_tiny", Path: "README.md", StartLine: 1},
			)},
			History: phase5DReplaceableHistory("call_tiny", "read_file", "tiny"),
		}

		result := agent.buildProviderHistoryProjection(ProviderHistoryReductionPolicy{Mode: ProviderHistoryReductionApply})

		if result.Report.CandidateCount != 1 || result.Report.ReplacedCount != 0 || result.Report.EstimatedSavedBytes != 0 {
			t.Fatalf("report = %#v, want skipped tiny candidate with no savings", result.Report)
		}
		assertKeepReason(t, result.Report, "call_tiny", "replacement_not_smaller")
	})
}

func phase5DToolRawContent(callID string, outputs map[string]string) string {
	if output, ok := outputs[callID]; ok {
		return output
	}
	switch callID {
	case "call_tail_read":
		return "latest trailing read output"
	case "call_tail_search":
		return "latest trailing search output"
	default:
		return ""
	}
}
