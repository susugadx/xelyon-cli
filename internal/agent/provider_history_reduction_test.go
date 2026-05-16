package agent

import (
	"reflect"
	"testing"
	"unicode/utf8"

	"github.com/susugadx/xelyon-cli/internal/api"
)

func TestProviderHistoryReductionDryRunDetectsOldAllowedToolResults(t *testing.T) {
	agent := &Agent{History: []api.Message{
		{Role: "user", Content: "inspect the repo"},
		providerHistoryAssistantToolCall("call_read_old", "read_file"),
		providerHistoryToolResult("call_read_old", "read_file", "old read_file output"),
		{Role: "assistant", Content: "I read it"},
		providerHistoryAssistantToolCalls(
			providerHistoryToolCall("call_search_old", "search_code"),
			providerHistoryToolCall("call_gather_old", "gather_context"),
		),
		providerHistoryToolResult("call_search_old", "", "old search_code output"),
		providerHistoryToolResult("call_gather_old", "gather_context", "old gather_context output"),
		{Role: "assistant", Content: "I have enough evidence"},
		providerHistoryAssistantToolCall("call_latest", "read_file"),
		providerHistoryToolResult("call_latest", "read_file", "latest read_file output"),
		{Role: "assistant", Content: "final answer"},
	}}

	result := agent.buildProviderHistoryProjection(ProviderHistoryReductionPolicy{Mode: ProviderHistoryReductionDryRun})

	if !reflect.DeepEqual(result.History, agent.History) {
		t.Fatalf("dry-run projection changed history:\n got %#v\nwant %#v", result.History, agent.History)
	}
	report := result.Report
	if report.Mode != ProviderHistoryReductionDryRun {
		t.Fatalf("report mode = %v, want dry-run", report.Mode)
	}
	if report.OriginalMessageCount != len(agent.History) || report.ProjectedMessageCount != len(agent.History) {
		t.Fatalf("message counts = (%d, %d), want %d", report.OriginalMessageCount, report.ProjectedMessageCount, len(agent.History))
	}
	if report.ToolResultCount != 4 {
		t.Fatalf("ToolResultCount = %d, want 4", report.ToolResultCount)
	}
	if got, want := candidateTools(report), []string{"read_file", "search_code", "gather_context"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("candidate tools = %#v, want %#v", got, want)
	}
	for _, candidate := range report.Candidates {
		if agent.History[candidate.HistoryIndex].Role != "tool" {
			t.Fatalf("candidate at history[%d] role = %q, want only tool results", candidate.HistoryIndex, agent.History[candidate.HistoryIndex].Role)
		}
		wantKind := "omit_old_" + candidate.ToolName + "_result"
		if candidate.SuggestedReplacementKind != wantKind {
			t.Fatalf("candidate %q replacement kind = %q, want %q", candidate.ToolName, candidate.SuggestedReplacementKind, wantKind)
		}
		if candidate.SuggestedReplacementText == "" {
			t.Fatalf("candidate %q missing suggested replacement text", candidate.ToolName)
		}
	}
	if latest := keptByToolCallID(report, "call_latest"); latest == nil || latest.KeepReason != "latest_tool_result" {
		t.Fatalf("latest tool result keep entry = %#v, want latest_tool_result", latest)
	}
}

func TestProviderHistoryReductionDryRunKeepsTrailingToolSuffix(t *testing.T) {
	agent := &Agent{History: []api.Message{
		providerHistoryAssistantToolCall("call_old", "read_file"),
		providerHistoryToolResult("call_old", "read_file", "old read"),
		{Role: "assistant", Content: "continue"},
		providerHistoryAssistantToolCalls(
			providerHistoryToolCall("call_tail_read", "read_file"),
			providerHistoryToolCall("call_tail_search", "search_code"),
		),
		providerHistoryToolResult("call_tail_read", "read_file", "tail read"),
		providerHistoryToolResult("call_tail_search", "search_code", "tail search"),
	}}

	report := agent.buildProviderHistoryProjection(ProviderHistoryReductionPolicy{Mode: ProviderHistoryReductionDryRun}).Report

	if got := candidateToolCallIDs(report); !reflect.DeepEqual(got, []string{"call_old"}) {
		t.Fatalf("candidate call IDs = %#v, want only old call", got)
	}
	for _, id := range []string{"call_tail_read", "call_tail_search"} {
		entry := keptByToolCallID(report, id)
		if entry == nil || entry.KeepReason != "trailing_tool_suffix" {
			t.Fatalf("trailing entry for %s = %#v, want trailing_tool_suffix", id, entry)
		}
	}
}

func TestProviderHistoryReductionDryRunAllowsRepeatedRescueIDsAcrossTurns(t *testing.T) {
	agent := &Agent{History: []api.Message{
		providerHistoryAssistantToolCall("call_rescue_001", "read_file"),
		providerHistoryToolResult("call_rescue_001", "read_file", "first rescue read"),
		{Role: "assistant", Content: "after first rescue read"},
		providerHistoryAssistantToolCall("call_rescue_001", "search_code"),
		providerHistoryToolResult("call_rescue_001", "", "second rescue search"),
		{Role: "assistant", Content: "after second rescue search"},
		providerHistoryAssistantToolCall("call_latest", "read_file"),
		providerHistoryToolResult("call_latest", "read_file", "latest"),
		{Role: "assistant", Content: "done"},
	}}

	report := agent.buildProviderHistoryProjection(ProviderHistoryReductionPolicy{Mode: ProviderHistoryReductionDryRun}).Report

	if got, want := candidateTools(report), []string{"read_file", "search_code"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("candidate tools = %#v, want repeated rescue calls resolved locally as %#v", got, want)
	}
	if got, want := candidateToolCallIDs(report), []string{"call_rescue_001", "call_rescue_001"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("candidate call IDs = %#v, want %#v", got, want)
	}
	for _, kept := range report.Kept {
		if kept.ToolCallID == "call_rescue_001" && kept.KeepReason == "ambiguous_tool_result_id" {
			t.Fatalf("repeated rescue ID across turns was kept as ambiguous: %#v", kept)
		}
	}
}

func TestProviderHistoryReductionDryRunKeepsWriteCommandAndNonAllowlistedTools(t *testing.T) {
	agent := &Agent{History: []api.Message{
		providerHistoryAssistantToolCall("call_patch", "apply_patch"),
		providerHistoryToolResult("call_patch", "apply_patch", "patched"),
		{Role: "assistant", Content: "patched"},
		providerHistoryAssistantToolCall("call_replace", "str_replace"),
		providerHistoryToolResult("call_replace", "str_replace", "replaced"),
		{Role: "assistant", Content: "replaced"},
		providerHistoryAssistantToolCall("call_bash", "bash"),
		providerHistoryToolResult("call_bash", "bash", "status"),
		{Role: "assistant", Content: "checked"},
		providerHistoryAssistantToolCall("call_list", "list_dir"),
		providerHistoryToolResult("call_list", "list_dir", "files"),
		{Role: "assistant", Content: "listed"},
		providerHistoryAssistantToolCall("call_latest", "read_file"),
		providerHistoryToolResult("call_latest", "read_file", "latest"),
		{Role: "assistant", Content: "done"},
	}}

	report := agent.buildProviderHistoryProjection(ProviderHistoryReductionPolicy{Mode: ProviderHistoryReductionDryRun}).Report

	if len(report.Candidates) != 0 {
		t.Fatalf("Candidates = %#v, want none", report.Candidates)
	}
	for _, id := range []string{"call_patch", "call_replace", "call_bash"} {
		entry := keptByToolCallID(report, id)
		if entry == nil || entry.KeepReason != "write_or_command_tool" {
			t.Fatalf("kept entry for %s = %#v, want write_or_command_tool", id, entry)
		}
	}
	entry := keptByToolCallID(report, "call_list")
	if entry == nil || entry.KeepReason != "tool_not_in_reduction_allowlist" {
		t.Fatalf("list_dir kept entry = %#v, want tool_not_in_reduction_allowlist", entry)
	}
}

func TestProviderHistoryReductionDryRunKeepsInvalidToolCallLinkage(t *testing.T) {
	agent := &Agent{History: []api.Message{
		{Role: "tool", ToolName: "read_file", Content: "missing id"},
		{Role: "assistant", Content: "after missing id"},
		providerHistoryToolResult("call_missing_assistant", "read_file", "missing assistant"),
		{Role: "assistant", Content: "after missing assistant"},
		providerHistoryAssistantToolCalls(
			providerHistoryToolCall("call_ambiguous_assistant", "read_file"),
			providerHistoryToolCall("call_ambiguous_assistant", "search_code"),
		),
		providerHistoryToolResult("call_ambiguous_assistant", "read_file", "ambiguous assistant"),
		{Role: "assistant", Content: "after ambiguous assistant"},
		providerHistoryAssistantToolCall("call_mismatch", "search_code"),
		providerHistoryToolResult("call_mismatch", "read_file", "mismatch"),
		{Role: "assistant", Content: "after mismatch"},
		providerHistoryAssistantToolCall("call_duplicate_result", "read_file"),
		providerHistoryToolResult("call_duplicate_result", "read_file", "duplicate 1"),
		providerHistoryToolResult("call_duplicate_result", "read_file", "duplicate 2"),
		{Role: "assistant", Content: "after duplicate 2"},
		providerHistoryAssistantToolCall("call_non_contiguous", "read_file"),
		{Role: "assistant", Content: "intervening assistant turn"},
		providerHistoryToolResult("call_non_contiguous", "read_file", "non-contiguous result"),
		{Role: "assistant", Content: "after non-contiguous result"},
		providerHistoryAssistantToolCall("call_latest", "read_file"),
		providerHistoryToolResult("call_latest", "read_file", "latest"),
		{Role: "assistant", Content: "done"},
	}}

	report := agent.buildProviderHistoryProjection(ProviderHistoryReductionPolicy{Mode: ProviderHistoryReductionDryRun}).Report

	if len(report.Candidates) != 0 {
		t.Fatalf("Candidates = %#v, want none", report.Candidates)
	}
	assertKeepReason(t, report, "", "missing_tool_call_id")
	assertKeepReason(t, report, "call_missing_assistant", "missing_assistant_tool_call")
	assertKeepReason(t, report, "call_ambiguous_assistant", "ambiguous_assistant_tool_call")
	assertKeepReason(t, report, "call_mismatch", "mismatched_tool_name")
	assertKeepReason(t, report, "call_duplicate_result", "ambiguous_tool_result_id")
	assertKeepReason(t, report, "call_non_contiguous", "non_contiguous_tool_call_linkage")
}

func TestProviderHistoryReductionDryRunMeasuresBytesAndRunes(t *testing.T) {
	content := "é日本a"
	agent := &Agent{History: []api.Message{
		providerHistoryAssistantToolCall("call_unicode", "read_file"),
		providerHistoryToolResult("call_unicode", "read_file", content),
		{Role: "assistant", Content: "after unicode"},
		providerHistoryAssistantToolCall("call_latest", "read_file"),
		providerHistoryToolResult("call_latest", "read_file", "latest"),
		{Role: "assistant", Content: "done"},
	}}

	report := agent.buildProviderHistoryProjection(ProviderHistoryReductionPolicy{Mode: ProviderHistoryReductionDryRun}).Report

	if len(report.Candidates) != 1 {
		t.Fatalf("Candidates = %#v, want one unicode candidate", report.Candidates)
	}
	candidate := report.Candidates[0]
	if candidate.OriginalByteSize != len(content) {
		t.Fatalf("OriginalByteSize = %d, want %d", candidate.OriginalByteSize, len(content))
	}
	if candidate.OriginalRuneSize != utf8.RuneCountInString(content) {
		t.Fatalf("OriginalRuneSize = %d, want %d", candidate.OriginalRuneSize, utf8.RuneCountInString(content))
	}
	if candidate.OriginalByteSize == candidate.OriginalRuneSize {
		t.Fatalf("byte and rune sizes should differ for %q: %#v", content, candidate)
	}
}

func TestProviderHistoryReductionDisabledReportEmptyAndProjectionNoOp(t *testing.T) {
	agent := &Agent{History: []api.Message{
		providerHistoryAssistantToolCall("call_old", "read_file"),
		providerHistoryToolResult("call_old", "read_file", "old read"),
		{Role: "assistant", Content: "after old read"},
	}}

	result := agent.buildProviderHistoryProjection(ProviderHistoryReductionPolicy{})

	if !reflect.DeepEqual(result.History, agent.History) {
		t.Fatalf("disabled projection = %#v, want raw history %#v", result.History, agent.History)
	}
	if !reflect.DeepEqual(result.Report, ProviderHistoryProjectionReport{}) {
		t.Fatalf("disabled report = %#v, want empty report", result.Report)
	}
}

func TestProviderHistoryReductionProjectionAndReportAreDefensiveCopies(t *testing.T) {
	agent := &Agent{History: []api.Message{
		providerHistoryAssistantToolCall("call_old", "read_file"),
		providerHistoryToolResult("call_old", "read_file", "old read"),
		{Role: "assistant", Content: "after old read"},
		providerHistoryAssistantToolCall("call_latest", "read_file"),
		providerHistoryToolResult("call_latest", "read_file", "latest"),
		{Role: "assistant", Content: "done"},
	}}

	result := agent.buildProviderHistoryProjection(ProviderHistoryReductionPolicy{Mode: ProviderHistoryReductionDryRun})
	if len(result.Report.Candidates) != 1 {
		t.Fatalf("Candidates = %#v, want one candidate", result.Report.Candidates)
	}

	result.History[0].ToolCalls[0].ID = "mutated_call"
	result.History[1].Content = "mutated content"
	result.Report.Candidates[0].ToolName = "mutated_tool"
	result.Report.Candidates[0].SuggestedReplacementText = "mutated replacement"

	if agent.History[0].ToolCalls[0].ID != "call_old" {
		t.Fatalf("Agent.History tool call ID = %q, want call_old", agent.History[0].ToolCalls[0].ID)
	}
	if agent.History[1].Content != "old read" {
		t.Fatalf("Agent.History tool content = %q, want old read", agent.History[1].Content)
	}
}

func candidateTools(report ProviderHistoryProjectionReport) []string {
	tools := make([]string, len(report.Candidates))
	for i, candidate := range report.Candidates {
		tools[i] = candidate.ToolName
	}
	return tools
}

func candidateToolCallIDs(report ProviderHistoryProjectionReport) []string {
	ids := make([]string, len(report.Candidates))
	for i, candidate := range report.Candidates {
		ids[i] = candidate.ToolCallID
	}
	return ids
}

func keptByToolCallID(report ProviderHistoryProjectionReport, id string) *ProviderHistoryReductionCandidate {
	for i := range report.Kept {
		if report.Kept[i].ToolCallID == id {
			return &report.Kept[i]
		}
	}
	return nil
}

func assertKeepReason(t *testing.T, report ProviderHistoryProjectionReport, id, want string) {
	t.Helper()
	entry := keptByToolCallID(report, id)
	if entry == nil || entry.KeepReason != want {
		t.Fatalf("kept entry for %q = %#v, want keep reason %q", id, entry, want)
	}
}
