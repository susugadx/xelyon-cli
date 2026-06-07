package agent

import (
	"reflect"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

func TestProviderHistoryReductionApplyReplacesAllowedToolResultsWithEvidencePointers(t *testing.T) {
	oldReadOutput := strings.Repeat("old read_file output\n", 240)
	oldSearchOutput := strings.Repeat("old search_code output\n", 240)
	oldGatherOutput := strings.Repeat("old gather_context output\n", 240)
	taskLedger := providerHistoryTaskLedgerWithEvidence(t,
		providerHistoryEvidenceItem{ToolName: "read_file", ToolCallID: "call_read_old", Path: "README.md", StartLine: 1, EndLine: 80},
		providerHistoryEvidenceItem{ToolName: "read_file", ToolCallID: "call_read_old", Path: "internal/a.go", StartLine: 2},
		providerHistoryEvidenceItem{ToolName: "read_file", ToolCallID: "call_read_old", Path: "internal/b.go", StartLine: 3, EndLine: 4},
		providerHistoryEvidenceItem{ToolName: "read_file", ToolCallID: "call_read_old", Path: "internal/c.go", StartLine: 5},
		providerHistoryEvidenceItem{ToolName: "read_file", ToolCallID: "call_read_old", Path: "internal/d.go", StartLine: 6},
		providerHistoryEvidenceItem{ToolName: "search_code", ToolCallID: "call_search_old", Path: "internal/search.go", StartLine: 9},
		providerHistoryEvidenceItem{ToolName: "gather_context", ToolCallID: "call_gather_old", Path: "internal/gather.go", StartLine: 10, EndLine: 12},
	)
	oldReadResult := providerHistoryToolResult("call_read_old", "read_file", oldReadOutput)
	oldReadResult.ReasoningContent = "tool reasoning"
	oldReadResult.SetAnthropicContentBlocks([]api.AnthropicContentBlock{{
		Type:  "tool_use",
		ID:    "toolu_read_old",
		Name:  "read_file",
		Input: map[string]any{"path": "README.md"},
	}})
	agent := &Agent{
		Runtime: &AgentRuntime{TaskLedger: taskLedger},
		History: []api.Message{
			{Role: "user", Content: "inspect the repo"},
			providerHistoryAssistantToolCall("call_read_old", "read_file"),
			oldReadResult,
			{Role: "assistant", Content: "I read it"},
			providerHistoryAssistantToolCalls(
				providerHistoryToolCall("call_search_old", "search_code"),
				providerHistoryToolCall("call_gather_old", "gather_context"),
			),
			providerHistoryToolResult("call_search_old", "", oldSearchOutput),
			providerHistoryToolResult("call_gather_old", "gather_context", oldGatherOutput),
			{Role: "assistant", Content: "I have enough evidence"},
			providerHistoryAssistantToolCall("call_latest", "read_file"),
			providerHistoryToolResult("call_latest", "read_file", "latest read_file output"),
			{Role: "assistant", Content: "final answer"},
		},
	}
	if agent.Runtime.Options.EnableCurrentTaskStateContext {
		t.Fatal("test setup unexpectedly enabled current task state context")
	}

	result := agent.buildProviderHistoryProjection(ProviderHistoryReductionPolicy{Mode: ProviderHistoryReductionApply})

	report := result.Report
	if report.Mode != ProviderHistoryReductionApply {
		t.Fatalf("report mode = %v, want apply", report.Mode)
	}
	if report.CandidateCount != 3 || report.ReplacedCount != 3 || report.ToolResultCount != 4 || report.KeptCount != 1 {
		t.Fatalf("report counts = candidates %d replaced %d toolResults %d kept %d, want 3/3/4/1", report.CandidateCount, report.ReplacedCount, report.ToolResultCount, report.KeptCount)
	}
	if !report.ResponsesChainDisabled {
		t.Fatalf("ResponsesChainDisabled = false, want true after provider-facing replacements")
	}
	wantRead := "[omitted old read_file result; evidence: README.md:L1-L80 source=read_file; internal/a.go:L2 source=read_file; internal/b.go:L3-L4 source=read_file; +2 more]"
	if result.History[2].Content != wantRead {
		t.Fatalf("read replacement = %q, want %q", result.History[2].Content, wantRead)
	}
	if result.History[5].Content != "[omitted old search_code result; evidence: internal/search.go:L9 source=search_code]" {
		t.Fatalf("search replacement = %q, want search_code placeholder", result.History[5].Content)
	}
	if result.History[6].Content != "[omitted old gather_context result; evidence: internal/gather.go:L10-L12 source=gather_context]" {
		t.Fatalf("gather replacement = %q, want gather_context placeholder", result.History[6].Content)
	}
	if result.History[2].Role != "tool" ||
		result.History[2].ToolCallID != "call_read_old" ||
		result.History[2].ToolName != "read_file" ||
		len(result.History[2].ToolCalls) != 0 {
		t.Fatalf("read replacement changed tool message shape: %#v", result.History[2])
	}
	if result.History[2].ReasoningContent != "tool reasoning" {
		t.Fatalf("read replacement ReasoningContent = %q, want preserved provider metadata", result.History[2].ReasoningContent)
	}
	if blocks := result.History[2].AnthropicContentBlocks(); len(blocks) != 1 ||
		blocks[0].ID != "toolu_read_old" ||
		blocks[0].Name != "read_file" ||
		blocks[0].Input["path"] != "README.md" {
		t.Fatalf("read replacement AnthropicContentBlocks = %#v, want preserved provider state", blocks)
	}
	if result.History[5].ToolCallID != "call_search_old" || result.History[5].ToolName != "search_code" {
		t.Fatalf("search replacement tool id/name = %#v, want inferred search_code", result.History[5])
	}
	if agent.History[5].ToolName != "" {
		t.Fatalf("raw search tool name = %q, want raw history unchanged", agent.History[5].ToolName)
	}
	for _, candidate := range report.Candidates {
		if !candidate.ReplacementApplied {
			t.Fatalf("candidate was not applied: %#v", candidate)
		}
		if len(candidate.EvidencePointers) == 0 {
			t.Fatalf("candidate missing matched evidence pointers after apply: %#v", candidate)
		}
		if result.History[candidate.HistoryIndex].Content != candidate.SuggestedReplacementText {
			t.Fatalf("candidate replacement text = %q, projection content = %q", candidate.SuggestedReplacementText, result.History[candidate.HistoryIndex].Content)
		}
	}
	if latest := keptByToolCallID(report, "call_latest"); latest == nil || latest.KeepReason != "latest_tool_result" {
		t.Fatalf("latest tool result keep entry = %#v, want latest_tool_result", latest)
	}
	if report.KeptReasonCounts["latest_tool_result"] != 1 {
		t.Fatalf("KeptReasonCounts = %#v, want latest_tool_result:1", report.KeptReasonCounts)
	}
	assertProviderHistoryByteMetrics(t, agent.History, result.History, report)
}

func TestProviderHistoryReductionApplyKeepsWhenReplacementWouldNotReduceBytes(t *testing.T) {
	taskLedger := providerHistoryTaskLedgerWithEvidence(t,
		providerHistoryEvidenceItem{ToolName: "read_file", ToolCallID: "call_tiny", Path: "README.md", StartLine: 1},
	)
	agent := &Agent{
		Runtime: &AgentRuntime{TaskLedger: taskLedger},
		History: []api.Message{
			providerHistoryAssistantToolCall("call_tiny", "read_file"),
			providerHistoryToolResult("call_tiny", "read_file", "tiny"),
			{Role: "assistant", Content: "after tiny read"},
			providerHistoryAssistantToolCall("call_latest", "read_file"),
			providerHistoryToolResult("call_latest", "read_file", "latest"),
			{Role: "assistant", Content: "done"},
		},
	}

	result := agent.buildProviderHistoryProjection(ProviderHistoryReductionPolicy{Mode: ProviderHistoryReductionApply})

	if !reflect.DeepEqual(result.History, agent.History) {
		t.Fatalf("apply projection with tiny candidate = %#v, want raw history %#v", result.History, agent.History)
	}
	candidate := candidateByToolCallID(result.Report, "call_tiny")
	if candidate == nil || candidate.ReplacementApplied || candidate.KeepReason != "replacement_not_smaller" {
		t.Fatalf("candidate = %#v, want replacement_not_smaller without replacement", candidate)
	}
	if candidate.SuggestedReplacementText != "[omitted old read_file result; evidence: README.md:L1 source=read_file]" {
		t.Fatalf("candidate replacement text = %q, want concrete skipped placeholder", candidate.SuggestedReplacementText)
	}
	assertKeepReason(t, result.Report, "call_tiny", "replacement_not_smaller")
	if result.Report.ReplacedCount != 0 || result.Report.KeptCount != result.Report.ToolResultCount || result.Report.EstimatedSavedBytes != 0 {
		t.Fatalf("report = replaced %d kept %d toolResults %d saved %d, want no increase/no replacement", result.Report.ReplacedCount, result.Report.KeptCount, result.Report.ToolResultCount, result.Report.EstimatedSavedBytes)
	}
	if result.Report.ResponsesChainDisabled {
		t.Fatalf("ResponsesChainDisabled = true, want false without replacement")
	}
	assertProviderHistoryByteMetrics(t, agent.History, result.History, result.Report)
}

func TestProviderHistoryReductionApplyKeepsCandidateWithoutEvidencePointer(t *testing.T) {
	agent := &Agent{History: []api.Message{
		providerHistoryAssistantToolCall("call_old", "read_file"),
		providerHistoryToolResult("call_old", "read_file", "old read"),
		{Role: "assistant", Content: "after old read"},
		providerHistoryAssistantToolCall("call_latest", "read_file"),
		providerHistoryToolResult("call_latest", "read_file", "latest"),
		{Role: "assistant", Content: "done"},
	}}

	result := agent.buildProviderHistoryProjection(ProviderHistoryReductionPolicy{Mode: ProviderHistoryReductionApply})

	if !reflect.DeepEqual(result.History, agent.History) {
		t.Fatalf("apply projection without evidence = %#v, want raw history %#v", result.History, agent.History)
	}
	candidate := candidateByToolCallID(result.Report, "call_old")
	if candidate == nil || candidate.ReplacementApplied || candidate.KeepReason != "missing_evidence_pointer" {
		t.Fatalf("candidate = %#v, want missing_evidence_pointer without replacement", candidate)
	}
	assertKeepReason(t, result.Report, "call_old", "missing_evidence_pointer")
	if result.Report.ReplacedCount != 0 || result.Report.KeptCount != result.Report.ToolResultCount {
		t.Fatalf("report replacement/keep counts = %d/%d with tool results %d, want 0/all kept", result.Report.ReplacedCount, result.Report.KeptCount, result.Report.ToolResultCount)
	}
	if result.Report.KeptReasonCounts["missing_evidence_pointer"] != 1 {
		t.Fatalf("KeptReasonCounts = %#v, want missing_evidence_pointer:1", result.Report.KeptReasonCounts)
	}
	if result.Report.ResponsesChainDisabled {
		t.Fatalf("ResponsesChainDisabled = true, want false without replacement")
	}
	assertProviderHistoryByteMetrics(t, agent.History, result.History, result.Report)
}

func TestProviderHistoryReductionApplyKeepsAmbiguousEvidencePointerCandidates(t *testing.T) {
	taskLedger := providerHistoryTaskLedgerWithEvidence(t,
		providerHistoryEvidenceItem{ToolName: "read_file", ToolCallID: "call_repeat", Path: "first.go", StartLine: 1},
	)
	agent := &Agent{
		Runtime: &AgentRuntime{TaskLedger: taskLedger},
		History: []api.Message{
			providerHistoryAssistantToolCall("call_repeat", "read_file"),
			providerHistoryToolResult("call_repeat", "read_file", "first repeated read"),
			{Role: "assistant", Content: "after first repeated read"},
			providerHistoryAssistantToolCall("call_repeat", "read_file"),
			providerHistoryToolResult("call_repeat", "read_file", "second repeated read"),
			{Role: "assistant", Content: "after second repeated read"},
			providerHistoryAssistantToolCall("call_latest", "read_file"),
			providerHistoryToolResult("call_latest", "read_file", "latest"),
			{Role: "assistant", Content: "done"},
		},
	}

	result := agent.buildProviderHistoryProjection(ProviderHistoryReductionPolicy{Mode: ProviderHistoryReductionApply})

	if !reflect.DeepEqual(result.History, agent.History) {
		t.Fatalf("ambiguous apply projection = %#v, want raw history %#v", result.History, agent.History)
	}
	if result.Report.CandidateCount != 2 || result.Report.ReplacedCount != 0 {
		t.Fatalf("candidate/replaced counts = %d/%d, want 2/0", result.Report.CandidateCount, result.Report.ReplacedCount)
	}
	if got := countKeptByToolCallIDAndReason(result.Report, "call_repeat", "ambiguous_evidence_pointer"); got != 2 {
		t.Fatalf("ambiguous keep count = %d, want 2: %#v", got, result.Report.Kept)
	}
	for _, candidate := range result.Report.Candidates {
		if candidate.ToolCallID == "call_repeat" && candidate.KeepReason != "ambiguous_evidence_pointer" {
			t.Fatalf("candidate = %#v, want ambiguous_evidence_pointer", candidate)
		}
	}
}

func TestProviderHistoryReductionApplyKeepsCandidateWhenEvidenceKeyReusedByTrailingToolResult(t *testing.T) {
	taskLedger := providerHistoryTaskLedgerWithEvidence(t,
		providerHistoryEvidenceItem{ToolName: "read_file", ToolCallID: "call_rescue_001", Path: "later.go", StartLine: 20},
	)
	oldReadOutput := strings.Repeat("old rescue read output\n", 8)
	agent := &Agent{
		Runtime: &AgentRuntime{TaskLedger: taskLedger},
		History: []api.Message{
			providerHistoryAssistantToolCall("call_rescue_001", "read_file"),
			providerHistoryToolResult("call_rescue_001", "read_file", oldReadOutput),
			{Role: "assistant", Content: "after old rescue read"},
			providerHistoryAssistantToolCall("call_rescue_001", "read_file"),
			providerHistoryToolResult("call_rescue_001", "read_file", "later trailing read"),
		},
	}

	result := agent.buildProviderHistoryProjection(ProviderHistoryReductionPolicy{Mode: ProviderHistoryReductionApply})

	if !reflect.DeepEqual(result.History, agent.History) {
		t.Fatalf("reused rescue key projection = %#v, want raw history %#v", result.History, agent.History)
	}
	if result.Report.CandidateCount != 1 || result.Report.ReplacedCount != 0 {
		t.Fatalf("candidate/replaced counts = %d/%d, want 1/0", result.Report.CandidateCount, result.Report.ReplacedCount)
	}
	assertKeepReason(t, result.Report, "call_rescue_001", "ambiguous_evidence_pointer")
	if got := countKeptByToolCallIDAndReason(result.Report, "call_rescue_001", "trailing_tool_suffix"); got != 1 {
		t.Fatalf("trailing keep count = %d, want 1: %#v", got, result.Report.Kept)
	}
	candidate := candidateByToolCallID(result.Report, "call_rescue_001")
	if candidate == nil || candidate.ReplacementApplied || candidate.KeepReason != "ambiguous_evidence_pointer" {
		t.Fatalf("candidate = %#v, want ambiguous evidence without replacement", candidate)
	}
}

func TestProviderHistoryReductionApplyKeepsWriteCommandTools(t *testing.T) {
	for _, toolName := range []string{"apply_patch", "str_replace", "bash", "write_file", "delete_file"} {
		t.Run(toolName, func(t *testing.T) {
			taskLedger := providerHistoryTaskLedgerWithEvidence(t,
				providerHistoryEvidenceItem{ToolName: toolName, ToolCallID: "call_write", Path: "changed.go", StartLine: 1},
			)
			agent := &Agent{
				Runtime: &AgentRuntime{TaskLedger: taskLedger},
				History: []api.Message{
					providerHistoryAssistantToolCall("call_write", toolName),
					providerHistoryToolResult("call_write", toolName, "write or command output"),
					{Role: "assistant", Content: "after write or command"},
					providerHistoryAssistantToolCall("call_latest", "read_file"),
					providerHistoryToolResult("call_latest", "read_file", "latest"),
					{Role: "assistant", Content: "done"},
				},
			}

			report := agent.buildProviderHistoryProjection(ProviderHistoryReductionPolicy{Mode: ProviderHistoryReductionApply}).Report

			if len(report.Candidates) != 0 || report.ReplacedCount != 0 {
				t.Fatalf("report candidates/replaced = %#v/%d, want no replacement for %s", report.Candidates, report.ReplacedCount, toolName)
			}
			assertKeepReason(t, report, "call_write", "write_or_command_tool")
		})
	}
}
