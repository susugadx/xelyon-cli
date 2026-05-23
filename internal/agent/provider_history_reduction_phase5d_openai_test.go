package agent

import (
	"context"
	"reflect"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	openairesponses "github.com/susugadx/xelyon-cli/internal/api/providers/openai_responses"
)

func TestPhase5DOpenAIResponsesReductionForcesFullHistoryAndKeepsTrailingToolOutputs(t *testing.T) {
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
	agent.Runtime.Options.EnableProviderHistoryReduction = true
	rawBefore := api.CloneMessages(agent.History)

	requestCtx, history := agent.providerFacingHistoryForRequest(context.Background())
	req := openairesponses.BuildChatRequest(openairesponses.ChatRequestOptions{
		Base:               openairesponses.BaseRequestOptions{Model: openairesponses.NewModelIdentity("gpt-5.4", ""), Store: true},
		RequestContext:     requestCtx,
		SystemPrompt:       "system",
		History:            history,
		PreviousResponseID: "resp_prev",
	})

	if history[2].Content == oldRead || !strings.HasPrefix(history[2].Content, providerHistoryReductionPlaceholderPrefix) {
		t.Fatalf("projected old read = %q, want reduction placeholder", history[2].Content)
	}
	if history[5].Content != trailingRead || history[6].Content != trailingSearch {
		t.Fatalf("trailing outputs = %q/%q, want raw suffix", history[5].Content, history[6].Content)
	}
	if !api.ResponseIDChainDisabledFromContext(requestCtx) {
		t.Fatal("request context did not disable response ID chain after replacement")
	}
	if req.PreviousResponseID != "" {
		t.Fatalf("PreviousResponseID = %q, want empty after replacement", req.PreviousResponseID)
	}
	items := phase5DResponsesInputItems(t, req.Input)
	oldItem := phase5DFindResponsesFunctionOutput(t, items, "call_old_read")
	if oldItem == nil || !strings.HasPrefix(oldItem.Output, providerHistoryReductionPlaceholderPrefix) {
		t.Fatalf("full input old read output = %#v, want reduction placeholder", oldItem)
	}
	tailReadItem := phase5DFindResponsesFunctionOutput(t, items, "call_tail_read")
	tailSearchItem := phase5DFindResponsesFunctionOutput(t, items, "call_tail_search")
	if tailReadItem == nil || tailSearchItem == nil {
		t.Fatalf("full input missing trailing function_call_output items: %#v", items)
	}
	phase5DAssertFunctionCallOutput(t, *tailReadItem, "call_tail_read", trailingRead)
	phase5DAssertFunctionCallOutput(t, *tailSearchItem, "call_tail_search", trailingSearch)
	if !reflect.DeepEqual(agent.History, rawBefore) {
		t.Fatalf("Agent.History changed after projection:\n got %#v\nwant %#v", agent.History, rawBefore)
	}
	report := agent.Runtime.LastProviderHistoryProjectionReport
	if report.CandidateCount != 1 || report.ReplacedCount != 1 || report.KeptCount != 2 {
		t.Fatalf("report counts = candidates %d replaced %d kept %d, want 1/1/2", report.CandidateCount, report.ReplacedCount, report.KeptCount)
	}
	if !report.ResponsesChainDisabled {
		t.Fatalf("ResponsesChainDisabled = false, want true after replacement")
	}
	if report.EstimatedSavedBytes <= 0 || !strings.Contains(formatProviderHistoryProjectionReportSummary(report), "saved=") {
		t.Fatalf("status projection report = %#v, want saved bytes for payload that contains placeholder", report)
	}
}

func TestPhase5DOpenAIResponsesContinuationKeptWhenProjectionHasNoReplacement(t *testing.T) {
	agent := &Agent{
		Runtime: &AgentRuntime{
			Options: RuntimeOptions{EnableProviderHistoryReduction: true},
		},
		History: phase5DReplaceableHistory("call_missing_evidence", "read_file", phase5DOutput("old read without evidence pointer")),
	}

	requestCtx, history := agent.providerFacingHistoryForRequest(context.Background())
	req := openairesponses.BuildChatRequest(openairesponses.ChatRequestOptions{
		Base:               openairesponses.BaseRequestOptions{Model: openairesponses.NewModelIdentity("gpt-5.4", ""), Store: true},
		RequestContext:     requestCtx,
		SystemPrompt:       "system",
		History:            history,
		PreviousResponseID: "resp_prev",
	})

	if api.ResponseIDChainDisabledFromContext(requestCtx) {
		t.Fatal("request context disabled response ID chain without replacement")
	}
	if req.PreviousResponseID != "resp_prev" {
		t.Fatalf("PreviousResponseID = %q, want resp_prev when ReplacedCount is zero", req.PreviousResponseID)
	}
	items := phase5DResponsesInputItems(t, req.Input)
	if len(items) != 1 {
		t.Fatalf("continuation input length = %d, want latest message only", len(items))
	}
	report := agent.Runtime.LastProviderHistoryProjectionReport
	if report.Mode != ProviderHistoryReductionApply || report.ReplacedCount != 0 || report.CandidateCount != 1 || report.ResponsesChainDisabled {
		t.Fatalf("report = %#v, want apply candidate without replacement or chain disable", report)
	}
}

func TestPhase5DOpenAIResponsesCommandReplacementDisablesContinuation(t *testing.T) {
	commandOutput := providerHistoryLargeSuccessfulTestOutput()
	agent := &Agent{
		Runtime: &AgentRuntime{
			Options: RuntimeOptions{EnableProviderHistoryReduction: true},
		},
		History: []api.Message{
			{Role: "user", Content: "run tests"},
			providerHistoryAssistantToolCalls(providerHistoryToolCallWithJSONArguments(t, "call_old_cmd", "bash", map[string]string{"command": providerHistorySuccessfulTestCommand})),
			providerHistoryToolResult("call_old_cmd", "bash", commandOutput),
			{Role: "assistant", Content: "tests passed"},
			providerHistoryAssistantToolCall("call_latest", "read_file"),
			providerHistoryToolResult("call_latest", "read_file", "latest raw output"),
			{Role: "assistant", Content: "done"},
		},
	}
	rawBefore := api.CloneMessages(agent.History)

	requestCtx, history := agent.providerFacingHistoryForRequest(context.Background())
	req := openairesponses.BuildChatRequest(openairesponses.ChatRequestOptions{
		Base:               openairesponses.BaseRequestOptions{Model: openairesponses.NewModelIdentity("gpt-5.4", ""), Store: true},
		RequestContext:     requestCtx,
		SystemPrompt:       "system",
		History:            history,
		PreviousResponseID: "resp_prev",
	})

	assertProviderHistoryCommandContentReplacement(t, history[2].Content, commandOutput, providerHistorySuccessfulTestReplacementLabel)
	if !api.ResponseIDChainDisabledFromContext(requestCtx) {
		t.Fatal("request context did not disable response ID chain after command replacement")
	}
	if req.PreviousResponseID != "" {
		t.Fatalf("PreviousResponseID = %q, want empty after command replacement", req.PreviousResponseID)
	}
	item := phase5DFindResponsesFunctionOutput(t, phase5DResponsesInputItems(t, req.Input), "call_old_cmd")
	if item == nil {
		t.Fatal("full input command output is missing")
	}
	assertProviderHistoryCommandContentReplacement(t, item.Output, commandOutput, providerHistorySuccessfulTestReplacementLabel)
	if !reflect.DeepEqual(agent.History, rawBefore) {
		t.Fatalf("Agent.History changed after command projection:\n got %#v\nwant %#v", agent.History, rawBefore)
	}
	report := agent.Runtime.LastProviderHistoryProjectionReport
	if report.ReplacedCount != 0 || report.CommandEditDryRun.CommandReplacedCount != 1 || !report.ResponsesChainDisabled {
		t.Fatalf("report = %#v, want command-only replacement with response chain disabled", report)
	}
}

func TestPhase5DOpenAIResponsesContinuationKeptWhenReductionDisabled(t *testing.T) {
	agent := &Agent{
		Runtime: &AgentRuntime{},
		History: []api.Message{
			providerHistoryAssistantToolCall("call_disabled", "read_file"),
			providerHistoryToolResult("call_disabled", "read_file", phase5DOutput("old read disabled")),
			{Role: "assistant", Content: "old read processed"},
			{Role: "user", Content: "next"},
		},
	}

	requestCtx, history := agent.providerFacingHistoryForRequest(context.Background())
	req := openairesponses.BuildChatRequest(openairesponses.ChatRequestOptions{
		Base:               openairesponses.BaseRequestOptions{Model: openairesponses.NewModelIdentity("gpt-5.4", ""), Store: true},
		RequestContext:     requestCtx,
		SystemPrompt:       "system",
		History:            history,
		PreviousResponseID: "resp_prev",
	})

	if api.ResponseIDChainDisabledFromContext(requestCtx) {
		t.Fatal("request context disabled response ID chain when reduction is disabled")
	}
	if req.PreviousResponseID != "resp_prev" {
		t.Fatalf("PreviousResponseID = %q, want resp_prev when reduction is disabled", req.PreviousResponseID)
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
	agent.Runtime.Options.EnableProviderHistoryReduction = true
	before := api.CloneMessages(agent.History)

	activeContext := []api.ActiveContextBlock{{
		Name:    "current_task_state",
		Content: "active context forces full input",
	}}
	requestCtx, history := agent.providerFacingHistoryForRequest(context.Background())
	requestCtx = api.WithActiveContextBlocks(requestCtx, activeContext)
	req := openairesponses.BuildChatRequest(openairesponses.ChatRequestOptions{
		Base:               openairesponses.BaseRequestOptions{Model: openairesponses.NewModelIdentity("gpt-5.4", ""), Store: true},
		RequestContext:     requestCtx,
		SystemPrompt:       "system",
		History:            history,
		PreviousResponseID: "resp_prev",
		ActiveContext:      activeContext,
	})

	if !api.ResponseIDChainDisabledFromContext(requestCtx) {
		t.Fatal("request context did not disable response ID chain after replacement")
	}
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
