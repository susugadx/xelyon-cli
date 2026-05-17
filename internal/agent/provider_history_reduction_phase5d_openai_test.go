package agent

import (
	"reflect"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	openairesponses "github.com/susugadx/xelyon-cli/internal/api/providers/openai_responses"
)

func TestPhase5DOpenAIResponsesContinuationKeepsTrailingToolOutputs(t *testing.T) {
	oldRead := phase5DOutput("old read_file continuation candidate")
	trailingRead := "fresh trailing read output"
	trailingSearch := "fresh trailing search output"
	agent := &Agent{
		Runtime: &AgentRuntime{TaskLedger: providerHistoryTaskLedgerWithEvidence(t,
			providerHistoryEvidenceItem{ToolName: "read_file", ToolCallID: "call_old_read", Path: "README.md", StartLine: 1, EndLine: 3},
		)},
		History: []api.Message{
			{Role: "user", Content: "inspect repo"},
			providerHistoryAssistantToolCall("call_old_read", "read_file"),
			providerHistoryToolResult("call_old_read", "read_file", oldRead),
			{Role: "assistant", Content: "old read processed"},
			providerHistoryAssistantToolCalls(
				providerHistoryToolCall("call_tail_read", "read_file"),
				providerHistoryToolCall("call_tail_search", "search_code"),
			),
			providerHistoryToolResult("call_tail_read", "read_file", trailingRead),
			providerHistoryToolResult("call_tail_search", "search_code", trailingSearch),
		},
	}

	result := agent.buildProviderHistoryProjection(ProviderHistoryReductionPolicy{Mode: ProviderHistoryReductionApply})
	req := openairesponses.BuildChatRequest(openairesponses.ChatRequestOptions{
		Base:               openairesponses.BaseRequestOptions{Model: openairesponses.NewModelIdentity("gpt-5.4", ""), Store: true},
		SystemPrompt:       "system",
		History:            result.History,
		PreviousResponseID: "resp_prev",
	})

	if result.History[2].Content == oldRead || !strings.HasPrefix(result.History[2].Content, providerHistoryReductionPlaceholderPrefix) {
		t.Fatalf("projected old read = %q, want reduction placeholder", result.History[2].Content)
	}
	if result.History[5].Content != trailingRead || result.History[6].Content != trailingSearch {
		t.Fatalf("trailing outputs = %q/%q, want raw suffix", result.History[5].Content, result.History[6].Content)
	}
	if agent.History[2].Content != oldRead {
		t.Fatalf("raw Agent.History old read = %q, want raw content", agent.History[2].Content)
	}
	if req.PreviousResponseID != "resp_prev" {
		t.Fatalf("PreviousResponseID = %q, want resp_prev", req.PreviousResponseID)
	}
	items := phase5DResponsesInputItems(t, req.Input)
	if len(items) != 2 {
		t.Fatalf("responses continuation input length = %d, want trailing tool suffix only: %#v", len(items), items)
	}
	phase5DAssertFunctionCallOutput(t, items[0], "call_tail_read", trailingRead)
	phase5DAssertFunctionCallOutput(t, items[1], "call_tail_search", trailingSearch)
	if strings.Contains(items[0].Output+items[1].Output, providerHistoryReductionPlaceholderPrefix) {
		t.Fatalf("continuation input contains old reduction placeholder: %#v", items)
	}
	report := result.Report
	if report.CandidateCount != 1 || report.ReplacedCount != 1 || report.KeptCount != 2 {
		t.Fatalf("report counts = candidates %d replaced %d kept %d, want 1/1/2", report.CandidateCount, report.ReplacedCount, report.KeptCount)
	}
}

func TestPhase5DOpenAIResponsesFullHistoryUsesProjectedInput(t *testing.T) {
	oldRead := phase5DOutput("old read_file full history candidate")
	rawHistory := []api.Message{
		{Role: "user", Content: "inspect repo"},
		providerHistoryAssistantToolCall("call_full_old", "read_file"),
		providerHistoryToolResult("call_full_old", "read_file", oldRead),
		{Role: "assistant", Content: "old read processed"},
		providerHistoryAssistantToolCall("call_latest", "read_file"),
		providerHistoryToolResult("call_latest", "read_file", "latest raw output"),
		{Role: "assistant", Content: "done"},
	}
	agent := &Agent{
		Runtime: &AgentRuntime{TaskLedger: providerHistoryTaskLedgerWithEvidence(t,
			providerHistoryEvidenceItem{ToolName: "read_file", ToolCallID: "call_full_old", Path: "README.md", StartLine: 5},
		)},
		History: rawHistory,
	}
	before := api.CloneMessages(agent.History)

	result := agent.buildProviderHistoryProjection(ProviderHistoryReductionPolicy{Mode: ProviderHistoryReductionApply})
	req := openairesponses.BuildChatRequest(openairesponses.ChatRequestOptions{
		Base:               openairesponses.BaseRequestOptions{Model: openairesponses.NewModelIdentity("gpt-5.4", ""), Store: true},
		SystemPrompt:       "system",
		History:            result.History,
		PreviousResponseID: "resp_prev",
		ActiveContext: []api.ActiveContextBlock{{
			Name:    "current_task_state",
			Content: "active context forces full input",
		}},
	})

	if req.PreviousResponseID != "" {
		t.Fatalf("PreviousResponseID = %q, want empty when active context forces full input", req.PreviousResponseID)
	}
	item := phase5DFindResponsesFunctionOutput(t, phase5DResponsesInputItems(t, req.Input), "call_full_old")
	if item == nil {
		t.Fatalf("full-history input missing function_call_output for call_full_old: %#v", req.Input)
	}
	if !strings.HasPrefix(item.Output, "[omitted old read_file result; evidence: README.md:L5 source=read_file]") {
		t.Fatalf("full-history projected output = %q, want read_file placeholder", item.Output)
	}
	if !reflect.DeepEqual(agent.History, before) {
		t.Fatalf("Agent.History changed after projection:\n got %#v\nwant %#v", agent.History, before)
	}
}

func phase5DResponsesInputItems(t *testing.T, input any) []openairesponses.InputItem {
	t.Helper()
	items, ok := input.([]openairesponses.InputItem)
	if !ok {
		t.Fatalf("responses input = %#v, want []InputItem", input)
	}
	return items
}

func phase5DAssertFunctionCallOutput(t *testing.T, item openairesponses.InputItem, callID, output string) {
	t.Helper()
	if item.Type != "function_call_output" || item.CallID != callID || item.Output != output {
		t.Fatalf("function_call_output = %#v, want call_id=%q output=%q", item, callID, output)
	}
}

func phase5DFindResponsesFunctionOutput(t *testing.T, items []openairesponses.InputItem, callID string) *openairesponses.InputItem {
	t.Helper()
	for i := range items {
		if items[i].Type == "function_call_output" && items[i].CallID == callID {
			return &items[i]
		}
	}
	return nil
}
