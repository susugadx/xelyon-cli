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
