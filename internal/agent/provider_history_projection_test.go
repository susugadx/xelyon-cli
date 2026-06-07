package agent

import (
	"reflect"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/history"
)

func TestProviderFacingHistory_NilAndEmptyReturnNil(t *testing.T) {
	var nilAgent *Agent
	if got := nilAgent.providerFacingHistory(); got != nil {
		t.Fatalf("nil agent providerFacingHistory() = %#v, want nil", got)
	}

	agent := &Agent{}
	if got := agent.providerFacingHistory(); got != nil {
		t.Fatalf("empty providerFacingHistory() = %#v, want nil", got)
	}
}

func TestProviderFacingHistory_ClonesRawRuntimeAndSessionState(t *testing.T) {
	assistant := api.Message{
		Role:    "assistant",
		Content: "I will inspect README.md",
		ToolCalls: []api.OpenAIToolCall{{
			ID:           "call_1",
			Type:         "function",
			Function:     api.OpenAIToolCallFunction{Name: "read_file", Arguments: `{"path":"README.md"}`},
			ThoughtParts: []map[string]any{{"text": "thinking"}},
		}},
	}
	assistant.SetAnthropicContentBlocks([]api.AnthropicContentBlock{{
		Type:  "tool_use",
		ID:    "toolu_1",
		Name:  "read_file",
		Input: map[string]any{"path": "README.md"},
	}})
	toolResult := api.Message{
		Role:       "tool",
		Content:    "README contents",
		ToolCallID: "call_1",
		ToolName:   "read_file",
	}

	agent := &Agent{
		CurrentModel: "test-model",
		History:      []api.Message{assistant, toolResult},
	}
	agent.session = history.NewSession("test-model")
	agent.session.AddMessageFromAPI(assistant, "test-model")
	agent.session.AddToolExecution("read_file", map[string]string{"path": "README.md"}, "README contents", true, "test-model")

	projection := agent.providerFacingHistory()
	if !reflect.DeepEqual(projection, agent.History) {
		t.Fatalf("projection = %#v, want same contents as Agent.History %#v", projection, agent.History)
	}
	if len(projection) != 2 || projection[0].Role != "assistant" || len(projection[0].ToolCalls) != 1 || projection[1].Role != "tool" {
		t.Fatalf("projection did not preserve tool call continuity: %#v", projection)
	}
	if projection[1].ToolCallID != projection[0].ToolCalls[0].ID {
		t.Fatalf("tool result ToolCallID = %q, want %q", projection[1].ToolCallID, projection[0].ToolCalls[0].ID)
	}

	projection[0].Content = "provider mutated"
	projection[0].ToolCalls[0].ID = "mutated_call"
	projection[0].ToolCalls[0].ThoughtParts[0]["text"] = "mutated thinking"
	projection[0].SetAnthropicContentBlocks([]api.AnthropicContentBlock{{Type: "tool_use", Input: map[string]any{"path": "mutated.go"}}})
	projection[1].Content = "mutated tool result"

	if agent.History[0].Content != "I will inspect README.md" {
		t.Fatalf("Agent.History[0].Content = %q, want original", agent.History[0].Content)
	}
	if agent.History[0].ToolCalls[0].ID != "call_1" {
		t.Fatalf("Agent.History tool call ID = %q, want call_1", agent.History[0].ToolCalls[0].ID)
	}
	if got := agent.History[0].ToolCalls[0].ThoughtParts[0]["text"]; got != "thinking" {
		t.Fatalf("Agent.History ThoughtParts text = %q, want thinking", got)
	}
	if agent.History[1].Content != "README contents" {
		t.Fatalf("Agent.History[1].Content = %q, want original tool result", agent.History[1].Content)
	}
	if agent.session.Messages[0].ToolCalls[0].ID != "call_1" {
		t.Fatalf("session conversation tool call ID = %q, want call_1", agent.session.Messages[0].ToolCalls[0].ID)
	}
	if agent.session.Messages[1].ToolExecution == nil || agent.session.Messages[1].ToolExecution.Name != "read_file" {
		t.Fatalf("session tool execution = %#v, want retained read_file audit entry", agent.session.Messages[1].ToolExecution)
	}
}

func TestProviderFacingHistoryExcludingLatestMessage(t *testing.T) {
	var nilAgent *Agent
	if got := nilAgent.providerFacingHistoryExcludingLatestMessage(); got != nil {
		t.Fatalf("nil agent providerFacingHistoryExcludingLatestMessage() = %#v, want nil", got)
	}

	agent := &Agent{History: []api.Message{{Role: "user", Content: "current prompt"}}}
	oneMessage := agent.providerFacingHistoryExcludingLatestMessage()
	if oneMessage == nil || len(oneMessage) != 0 {
		t.Fatalf("single-message providerFacingHistoryExcludingLatestMessage() = %#v, want non-nil empty history", oneMessage)
	}

	agent.History = []api.Message{
		{Role: "assistant", Content: "previous context"},
		{Role: "user", Content: "current prompt"},
	}
	got := agent.providerFacingHistoryExcludingLatestMessage()
	if len(got) != 1 || got[0].Content != "previous context" {
		t.Fatalf("providerFacingHistoryExcludingLatestMessage() = %#v, want previous context only", got)
	}
	got[0].Content = "provider mutated"
	if agent.History[0].Content != "previous context" {
		t.Fatalf("Agent.History[0].Content = %q, want previous context", agent.History[0].Content)
	}
}

func TestProviderFacingHistoryRecordsEmptyReportWhenRuntimeGateDisabled(t *testing.T) {
	runtime := &AgentRuntime{
		LastProviderHistoryProjectionReport: ProviderHistoryProjectionReport{
			Mode:                ProviderHistoryReductionApply,
			CandidateCount:      1,
			ReplacedCount:       1,
			EstimatedSavedBytes: 100,
		},
	}
	agent := &Agent{
		Runtime: runtime,
		History: []api.Message{
			providerHistoryAssistantToolCall("call_old", "read_file"),
			providerHistoryToolResult("call_old", "read_file", strings.Repeat("old read\n", 12)),
			{Role: "assistant", Content: "after old read"},
		},
	}

	projection := agent.providerFacingHistory()

	if !reflect.DeepEqual(projection, agent.History) {
		t.Fatalf("disabled providerFacingHistory() = %#v, want raw clone %#v", projection, agent.History)
	}
	if !reflect.DeepEqual(runtime.LastProviderHistoryProjectionReport, ProviderHistoryProjectionReport{}) {
		t.Fatalf("LastProviderHistoryProjectionReport = %#v, want empty disabled report", runtime.LastProviderHistoryProjectionReport)
	}
}

func TestProviderFacingHistoryAppliesRuntimeGateAndRecordsReport(t *testing.T) {
	oldRead := strings.Repeat("old read_file output\n", 240)
	runtime := &AgentRuntime{
		Options: RuntimeOptions{EnableProviderHistoryReduction: true},
		TaskLedger: providerHistoryTaskLedgerWithEvidence(t,
			providerHistoryEvidenceItem{ToolName: "read_file", ToolCallID: "call_old", Path: "README.md", StartLine: 1, EndLine: 2},
		),
	}
	agent := &Agent{
		Runtime: runtime,
		History: []api.Message{
			providerHistoryAssistantToolCall("call_old", "read_file"),
			providerHistoryToolResult("call_old", "read_file", oldRead),
			{Role: "assistant", Content: "after old read"},
			providerHistoryAssistantToolCall("call_latest", "read_file"),
			providerHistoryToolResult("call_latest", "read_file", "latest read"),
			{Role: "assistant", Content: "done"},
		},
	}

	projection := agent.providerFacingHistory()

	if projection[1].Content != "[omitted old read_file result; evidence: README.md:L1-L2 source=read_file]" {
		t.Fatalf("projection old tool content = %q, want provider placeholder", projection[1].Content)
	}
	if agent.History[1].Content != oldRead {
		t.Fatalf("Agent.History[1].Content = %q, want raw old read", agent.History[1].Content)
	}
	report := runtime.LastProviderHistoryProjectionReport
	if report.Mode != ProviderHistoryReductionApply || report.ReplacedCount != 1 || report.EstimatedSavedBytes <= 0 {
		t.Fatalf("LastProviderHistoryProjectionReport = %#v, want apply report with one replacement and saved bytes", report)
	}
}

func TestProviderFacingHistoryAppliesRuntimeGateAndPreservesInferredToolName(t *testing.T) {
	oldRead := strings.Repeat("old read_file output without stored tool name\n", 240)
	runtime := &AgentRuntime{
		Options: RuntimeOptions{EnableProviderHistoryReduction: true},
		TaskLedger: providerHistoryTaskLedgerWithEvidence(t,
			providerHistoryEvidenceItem{ToolName: "read_file", ToolCallID: "call_old", Path: "README.md", StartLine: 1},
		),
	}
	agent := &Agent{
		Runtime: runtime,
		History: []api.Message{
			providerHistoryAssistantToolCall("call_old", "read_file"),
			{Role: "tool", ToolCallID: "call_old", Content: oldRead},
			{Role: "assistant", Content: "after old read"},
			providerHistoryAssistantToolCall("call_latest", "read_file"),
			providerHistoryToolResult("call_latest", "read_file", "latest read"),
			{Role: "assistant", Content: "done"},
		},
	}

	projection := agent.providerFacingHistory()

	if projection[1].Content != "[omitted old read_file result; evidence: README.md:L1 source=read_file]" {
		t.Fatalf("projection old tool content = %q, want provider placeholder", projection[1].Content)
	}
	if projection[1].ToolName != "read_file" {
		t.Fatalf("projection old tool name = %q, want inferred read_file", projection[1].ToolName)
	}
	if agent.History[1].ToolName != "" {
		t.Fatalf("Agent.History[1].ToolName = %q, want raw history unchanged", agent.History[1].ToolName)
	}
	report := runtime.LastProviderHistoryProjectionReport
	candidate := candidateByToolCallID(report, "call_old")
	if candidate == nil || candidate.ToolName != "read_file" || !candidate.ReplacementApplied {
		t.Fatalf("candidate = %#v, want inferred read_file replacement", candidate)
	}
}

func TestProviderFacingHistoryExcludingLatestMessageReportsProjectedPastHistory(t *testing.T) {
	oldRead := strings.Repeat("old image read output\n", 240)
	runtime := &AgentRuntime{
		Options: RuntimeOptions{EnableProviderHistoryReduction: true},
		TaskLedger: providerHistoryTaskLedgerWithEvidence(t,
			providerHistoryEvidenceItem{ToolName: "read_file", ToolCallID: "call_image_old", Path: "image.md", StartLine: 4},
		),
	}
	agent := &Agent{
		Runtime: runtime,
		History: []api.Message{
			providerHistoryAssistantToolCall("call_image_old", "read_file"),
			providerHistoryToolResult("call_image_old", "read_file", oldRead),
			{Role: "assistant", Content: "after image read"},
			providerHistoryAssistantToolCall("call_image_latest", "read_file"),
			providerHistoryToolResult("call_image_latest", "read_file", "latest image read"),
			{Role: "assistant", Content: "after latest image read"},
			{Role: "user", Content: "current image prompt"},
		},
	}

	projection := agent.providerFacingHistoryExcludingLatestMessage()

	if len(projection) != 6 {
		t.Fatalf("excluding projection length = %d, want past history only", len(projection))
	}
	if projection[1].Content != "[omitted old read_file result; evidence: image.md:L4 source=read_file]" {
		t.Fatalf("excluding old tool content = %q, want provider placeholder", projection[1].Content)
	}
	report := runtime.LastProviderHistoryProjectionReport
	if report.OriginalMessageCount != 6 || report.ProjectedMessageCount != 6 || report.ReplacedCount != 1 {
		t.Fatalf("excluding report = %#v, want counts for provider past history only", report)
	}
}
