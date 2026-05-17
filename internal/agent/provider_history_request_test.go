package agent

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

func TestNormalModeRequestUsesProviderFacingHistoryClone(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &providerFacingHistoryMutationProbe{}
	agent := newChatRequestTestAgent(t, provider, &out)
	agent.History = []api.Message{{
		Role:    "assistant",
		Content: "previous assistant",
		ToolCalls: []api.OpenAIToolCall{{
			ID:           "call_1",
			Type:         "function",
			Function:     api.OpenAIToolCallFunction{Name: "read_file", Arguments: `{"path":"README.md"}`},
			ThoughtParts: []map[string]any{{"text": "thinking"}},
		}},
	}, {
		Role:       "tool",
		Content:    "old read_file reduction candidate",
		ToolCallID: "call_1",
		ToolName:   "read_file",
	}, {
		Role:    "assistant",
		Content: "previous assistant",
	}}

	if err := agent.chatInternal("next request", nil); err != nil {
		t.Fatalf("chatInternal() error = %v", err)
	}

	if len(provider.capturedHistory) != 4 {
		t.Fatalf("provider history length = %d, want previous assistant/tool context + current user", len(provider.capturedHistory))
	}
	if provider.capturedHistory[0].Content != "previous assistant" {
		t.Fatalf("provider first history content = %q, want previous assistant", provider.capturedHistory[0].Content)
	}
	if provider.capturedHistory[1].Content != "old read_file reduction candidate" {
		t.Fatalf("provider old tool result = %q, want unchanged reduction candidate content", provider.capturedHistory[1].Content)
	}
	if provider.capturedHistory[2].Content != "previous assistant" {
		t.Fatalf("provider previous assistant content = %q, want previous assistant", provider.capturedHistory[2].Content)
	}
	if !strings.Contains(provider.capturedHistory[3].Content, "next request") {
		t.Fatalf("provider current user content = %q, want current request", provider.capturedHistory[3].Content)
	}
	if agent.History[0].Content != "previous assistant" {
		t.Fatalf("Agent.History[0].Content = %q, want previous assistant", agent.History[0].Content)
	}
	if agent.History[0].ToolCalls[0].ID != "call_1" {
		t.Fatalf("Agent.History tool call ID = %q, want call_1", agent.History[0].ToolCalls[0].ID)
	}
	if got := agent.History[0].ToolCalls[0].ThoughtParts[0]["text"]; got != "thinking" {
		t.Fatalf("Agent.History ThoughtParts text = %q, want thinking", got)
	}
}

func TestImageRequestUsesProjectedPastHistoryAndCurrentPrompt(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &providerFacingHistoryMutationProbe{supportsImages: true}
	agent := newChatRequestTestAgent(t, provider, &out)
	agent.History = []api.Message{
		providerHistoryAssistantToolCall("call_image_old", "read_file"),
		providerHistoryToolResult("call_image_old", "read_file", "old image-history read_file result"),
		{Role: "assistant", Content: "previous image context"},
	}
	image := &api.ImageData{Base64: "dGVzdA==", MediaType: "image/png", Path: "test.png", Size: 4}

	if err := agent.chatInternal("describe image", image); err != nil {
		t.Fatalf("chatInternal() error = %v", err)
	}

	if provider.imageCalls != 1 {
		t.Fatalf("ChatWithImage calls = %d, want 1", provider.imageCalls)
	}
	if len(provider.capturedHistory) != 3 {
		t.Fatalf("image provider history length = %d, want past history only", len(provider.capturedHistory))
	}
	if provider.capturedHistory[1].Content != "old image-history read_file result" {
		t.Fatalf("image provider old tool result = %q, want unchanged reduction candidate content", provider.capturedHistory[1].Content)
	}
	if provider.capturedHistory[2].Content != "previous image context" {
		t.Fatalf("image provider history[2] = %q, want previous image context", provider.capturedHistory[2].Content)
	}
	if !strings.Contains(provider.imageUserMessage, "describe image") || !strings.Contains(provider.imageUserMessage, "[NORMAL MODE]") {
		t.Fatalf("image userMessage = %q, want current prompt with normal-mode directive", provider.imageUserMessage)
	}
	if agent.History[1].Content != "old image-history read_file result" {
		t.Fatalf("Agent.History[1].Content = %q, want old image-history read_file result", agent.History[1].Content)
	}
	if agent.History[2].Content != "previous image context" {
		t.Fatalf("Agent.History[2].Content = %q, want previous image context", agent.History[2].Content)
	}
}

func TestHeadlessRequestUsesProviderFacingHistoryClone(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	provider := &providerFacingHistoryMutationProbe{}
	runner := newHeadlessRunner("headless query", "test-model", provider, newProjectMapDisabledConfig())
	t.Cleanup(runner.agent.Cleanup)
	runner.agent.History = []api.Message{
		providerHistoryAssistantToolCall("call_headless_old", "read_file"),
		providerHistoryToolResult("call_headless_old", "read_file", "old headless read_file result"),
		{Role: "assistant", Content: "previous headless context"},
		{Role: "user", Content: "headless query"},
	}

	if _, err := runner.requestAssistantResponse(context.Background(), 0); err != nil {
		t.Fatalf("requestAssistantResponse() error = %v", err)
	}

	if len(provider.capturedHistory) != 4 || provider.capturedHistory[1].Content != "old headless read_file result" || provider.capturedHistory[3].Content != "headless query" {
		t.Fatalf("headless provider history = %#v, want raw previous context and query", provider.capturedHistory)
	}
	if runner.agent.History[1].Content != "old headless read_file result" {
		t.Fatalf("headless Agent.History[1].Content = %q, want old tool result", runner.agent.History[1].Content)
	}
}

func TestPlanInvestigationRequestUsesProviderFacingHistoryClone(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &providerFacingHistoryMutationProbe{response: "investigation done"}
	agent := newChatRequestTestAgent(t, provider, &out)
	agent.History = []api.Message{
		providerHistoryAssistantToolCall("call_plan_old", "read_file"),
		providerHistoryToolResult("call_plan_old", "read_file", "old plan read_file result"),
		{Role: "assistant", Content: "previous plan context"},
		{Role: "user", Content: "investigation prompt"},
	}

	response, err := newPlanInvestigationRunner(agent, context.Background()).requestResponse()
	if err != nil {
		t.Fatalf("requestResponse() error = %v", err)
	}
	if response != "investigation done" {
		t.Fatalf("response = %q, want investigation done", response)
	}
	if len(provider.capturedHistory) != 4 || provider.capturedHistory[1].Content != "old plan read_file result" || provider.capturedHistory[3].Content != "investigation prompt" {
		t.Fatalf("plan provider history = %#v, want raw previous context and investigation prompt", provider.capturedHistory)
	}
	if agent.History[1].Content != "old plan read_file result" {
		t.Fatalf("plan Agent.History[1].Content = %q, want old tool result", agent.History[1].Content)
	}
}

func TestNormalModeRequestAppliesProviderHistoryReductionWhenRuntimeGateEnabled(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &providerFacingHistoryMutationProbe{}
	agent := newChatRequestTestAgent(t, provider, &out)
	oldRead := seedProviderHistoryReductionRequestFixture(t, agent, "call_normal_old")
	agent.session.AddMessageFromAPI(agent.History[0], agent.CurrentModel)
	agent.session.AddMessageFromAPI(agent.History[1], agent.CurrentModel)
	agent.session.AddToolExecution("read_file", map[string]string{"path": "README.md"}, oldRead, true, agent.CurrentModel)

	if err := agent.chatInternal("next request", nil); err != nil {
		t.Fatalf("chatInternal() error = %v", err)
	}

	assertProviderRequestHistoryReductionApplied(t, agent, provider, oldRead, "next request")
	if agent.session.Messages[1].Content != oldRead {
		t.Fatalf("session conversation tool content = %q, want raw old read", agent.session.Messages[1].Content)
	}
	if agent.session.Messages[2].ToolExecution == nil || agent.session.Messages[2].ToolExecution.ResultPreview != oldRead {
		t.Fatalf("session tool execution = %#v, want raw old read audit entry", agent.session.Messages[2].ToolExecution)
	}
}

func TestNormalModeRequestKeepsReductionCandidateWithoutEvidencePointer(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &providerFacingHistoryMutationProbe{}
	agent := newChatRequestTestAgent(t, provider, &out)
	oldRead := seedProviderHistoryReductionRequestFixtureWithoutEvidence(agent, "call_missing_evidence")

	if err := agent.chatInternal("next request", nil); err != nil {
		t.Fatalf("chatInternal() error = %v", err)
	}

	if provider.capturedHistory[1].Content != oldRead {
		t.Fatalf("provider old tool result = %q, want raw candidate without evidence", provider.capturedHistory[1].Content)
	}
	if agent.History[1].Content != oldRead {
		t.Fatalf("Agent.History[1].Content = %q, want raw old read", agent.History[1].Content)
	}
	report := agent.Runtime.LastProviderHistoryProjectionReport
	if report.Mode != ProviderHistoryReductionApply || report.CandidateCount != 1 || report.ReplacedCount != 0 {
		t.Fatalf("LastProviderHistoryProjectionReport = %#v, want one unapplied apply candidate", report)
	}
	candidate := candidateByToolCallID(report, "call_missing_evidence")
	if candidate == nil || candidate.ReplacementApplied || candidate.KeepReason != "missing_evidence_pointer" {
		t.Fatalf("candidate = %#v, want missing_evidence_pointer without replacement", candidate)
	}
}

func TestNormalModeRequestAppliesProviderHistoryReductionPreservesInferredToolName(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &providerFacingHistoryMutationProbe{name: "gemini"}
	agent := newChatRequestTestAgent(t, provider, &out)
	oldRead := seedProviderHistoryReductionRequestFixture(t, agent, "call_missing_stored_name")
	agent.History[1].ToolName = ""

	if err := agent.chatInternal("next request", nil); err != nil {
		t.Fatalf("chatInternal() error = %v", err)
	}

	assertProviderRequestHistoryReductionApplied(t, agent, provider, oldRead, "next request")
	if provider.capturedHistory[1].ToolName != "read_file" {
		t.Fatalf("provider old tool name = %q, want inferred read_file for reduced tool response", provider.capturedHistory[1].ToolName)
	}
	if agent.History[1].ToolName != "" {
		t.Fatalf("Agent.History[1].ToolName = %q, want raw history unchanged", agent.History[1].ToolName)
	}
}

func TestImageRequestAppliesProviderHistoryReductionToPastHistoryWhenRuntimeGateEnabled(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &providerFacingHistoryMutationProbe{supportsImages: true}
	agent := newChatRequestTestAgent(t, provider, &out)
	oldRead := seedProviderHistoryReductionRequestFixture(t, agent, "call_image_apply_old")
	image := &api.ImageData{Base64: "dGVzdA==", MediaType: "image/png", Path: "test.png", Size: 4}

	if err := agent.chatInternal("describe image", image); err != nil {
		t.Fatalf("chatInternal() error = %v", err)
	}

	assertProviderRequestHistoryReductionApplied(t, agent, provider, oldRead, "")
	if len(provider.capturedHistory) != 6 {
		t.Fatalf("image provider history length = %d, want only projected past history", len(provider.capturedHistory))
	}
	if strings.Contains(strings.Join(providerHistoryMessageContents(provider.capturedHistory), "\n"), "describe image") {
		t.Fatalf("image provider history should exclude current prompt, got %#v", provider.capturedHistory)
	}
	if !strings.Contains(provider.imageUserMessage, "describe image") {
		t.Fatalf("image userMessage = %q, want current prompt", provider.imageUserMessage)
	}
}

func TestHeadlessRequestAppliesProviderHistoryReductionWhenRuntimeGateEnabled(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	provider := &providerFacingHistoryMutationProbe{}
	runner := newHeadlessRunner("headless query", "test-model", provider, newProjectMapDisabledConfig())
	t.Cleanup(runner.agent.Cleanup)
	oldRead := seedProviderHistoryReductionRequestFixture(t, runner.agent, "call_headless_apply_old")

	if _, err := runner.requestAssistantResponse(context.Background(), 0); err != nil {
		t.Fatalf("requestAssistantResponse() error = %v", err)
	}

	assertProviderRequestHistoryReductionApplied(t, runner.agent, provider, oldRead, "")
}

func TestPlanInvestigationRequestAppliesProviderHistoryReductionWhenRuntimeGateEnabled(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &providerFacingHistoryMutationProbe{response: "investigation done"}
	agent := newChatRequestTestAgent(t, provider, &out)
	oldRead := seedProviderHistoryReductionRequestFixture(t, agent, "call_plan_apply_old")

	response, err := newPlanInvestigationRunner(agent, context.Background()).requestResponse()
	if err != nil {
		t.Fatalf("requestResponse() error = %v", err)
	}
	if response != "investigation done" {
		t.Fatalf("response = %q, want investigation done", response)
	}
	assertProviderRequestHistoryReductionApplied(t, agent, provider, oldRead, "")
}
