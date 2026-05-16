package agent

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

type providerFacingHistoryMutationProbe struct {
	name             string
	supportsImages   bool
	response         string
	capturedContents []string
	capturedLen      int
	imageUserMessage string
	imageCalls       int
}

func (p *providerFacingHistoryMutationProbe) Name() string {
	if p.name != "" {
		return p.name
	}
	return "openai"
}

func (p *providerFacingHistoryMutationProbe) SupportsImages() bool { return p.supportsImages }

func (p *providerFacingHistoryMutationProbe) IsFunctionCallingEnabled() bool { return true }

func (p *providerFacingHistoryMutationProbe) ChatWithTools(_ context.Context, _ string, history []api.Message, _ string) (string, error) {
	p.capture(history)
	mutateProviderFacingHistoryForTest(history)
	if p.response != "" {
		return p.response, nil
	}
	return "provider response", nil
}

func (p *providerFacingHistoryMutationProbe) ChatWithImage(_ context.Context, _ string, history []api.Message, userMessage string, image *api.ImageData, _ string) (string, error) {
	if image == nil {
		return "", fmt.Errorf("image is required")
	}
	p.imageCalls++
	p.imageUserMessage = userMessage
	p.capture(history)
	mutateProviderFacingHistoryForTest(history)
	if p.response != "" {
		return p.response, nil
	}
	return "image response", nil
}

func (p *providerFacingHistoryMutationProbe) capture(messages []api.Message) {
	p.capturedLen = len(messages)
	p.capturedContents = make([]string, len(messages))
	for i, msg := range messages {
		p.capturedContents[i] = msg.Content
	}
}

func mutateProviderFacingHistoryForTest(messages []api.Message) {
	if len(messages) == 0 {
		return
	}
	messages[0].Content = "provider mutated content"
	if len(messages[0].ToolCalls) > 0 {
		messages[0].ToolCalls[0].ID = "provider_mutated_call"
		if len(messages[0].ToolCalls[0].ThoughtParts) > 0 {
			messages[0].ToolCalls[0].ThoughtParts[0]["text"] = "provider mutated thought"
		}
	}
}

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
	}}

	if err := agent.chatInternal("next request", nil); err != nil {
		t.Fatalf("chatInternal() error = %v", err)
	}

	if provider.capturedLen != 2 {
		t.Fatalf("provider history length = %d, want previous + current user", provider.capturedLen)
	}
	if provider.capturedContents[0] != "previous assistant" {
		t.Fatalf("provider first history content = %q, want previous assistant", provider.capturedContents[0])
	}
	if !strings.Contains(provider.capturedContents[1], "next request") {
		t.Fatalf("provider current user content = %q, want current request", provider.capturedContents[1])
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
	agent.History = []api.Message{{Role: "assistant", Content: "previous image context"}}
	image := &api.ImageData{Base64: "dGVzdA==", MediaType: "image/png", Path: "test.png", Size: 4}

	if err := agent.chatInternal("describe image", image); err != nil {
		t.Fatalf("chatInternal() error = %v", err)
	}

	if provider.imageCalls != 1 {
		t.Fatalf("ChatWithImage calls = %d, want 1", provider.imageCalls)
	}
	if provider.capturedLen != 1 {
		t.Fatalf("image provider history length = %d, want past history only", provider.capturedLen)
	}
	if provider.capturedContents[0] != "previous image context" {
		t.Fatalf("image provider history[0] = %q, want previous image context", provider.capturedContents[0])
	}
	if !strings.Contains(provider.imageUserMessage, "describe image") || !strings.Contains(provider.imageUserMessage, "[NORMAL MODE]") {
		t.Fatalf("image userMessage = %q, want current prompt with normal-mode directive", provider.imageUserMessage)
	}
	if agent.History[0].Content != "previous image context" {
		t.Fatalf("Agent.History[0].Content = %q, want previous image context", agent.History[0].Content)
	}
}

func TestHeadlessRequestUsesProviderFacingHistoryClone(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	provider := &providerFacingHistoryMutationProbe{}
	runner := newHeadlessRunner("headless query", "test-model", provider, newProjectMapDisabledConfig())
	t.Cleanup(runner.agent.Cleanup)

	if _, err := runner.requestAssistantResponse(context.Background(), 0); err != nil {
		t.Fatalf("requestAssistantResponse() error = %v", err)
	}

	if provider.capturedLen != 1 || provider.capturedContents[0] != "headless query" {
		t.Fatalf("headless provider history = len %d contents %#v, want raw query", provider.capturedLen, provider.capturedContents)
	}
	if runner.agent.History[0].Content != "headless query" {
		t.Fatalf("headless Agent.History[0].Content = %q, want headless query", runner.agent.History[0].Content)
	}
}

func TestPlanInvestigationRequestUsesProviderFacingHistoryClone(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	provider := &providerFacingHistoryMutationProbe{response: "investigation done"}
	agent := newChatRequestTestAgent(t, provider, &out)
	agent.History = []api.Message{{Role: "user", Content: "investigation prompt"}}

	response, err := newPlanInvestigationRunner(agent, context.Background()).requestResponse()
	if err != nil {
		t.Fatalf("requestResponse() error = %v", err)
	}
	if response != "investigation done" {
		t.Fatalf("response = %q, want investigation done", response)
	}
	if provider.capturedLen != 1 || provider.capturedContents[0] != "investigation prompt" {
		t.Fatalf("plan provider history = len %d contents %#v, want investigation prompt", provider.capturedLen, provider.capturedContents)
	}
	if agent.History[0].Content != "investigation prompt" {
		t.Fatalf("plan Agent.History[0].Content = %q, want investigation prompt", agent.History[0].Content)
	}
}
