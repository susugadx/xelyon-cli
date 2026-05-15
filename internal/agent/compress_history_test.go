package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

type capturingMockProvider struct {
	capturedHistory []api.Message
}

func (m *capturingMockProvider) Name() string                   { return "test" }
func (m *capturingMockProvider) SupportsImages() bool           { return false }
func (m *capturingMockProvider) IsFunctionCallingEnabled() bool { return true }
func (m *capturingMockProvider) ChatWithTools(_ context.Context, _ string, history []api.Message, _ string) (string, error) {
	m.capturedHistory = history
	return "Summary of conversation", nil
}
func (m *capturingMockProvider) ChatWithImage(_ context.Context, _ string, _ []api.Message, _ string, _ *api.ImageData, _ string) (string, error) {
	return "", nil
}

func TestCompressHistory_PrePrunesBeforeSummary(t *testing.T) {
	provider := &capturingMockProvider{}
	agent := NewAgent("test-model", provider, false)

	large := makeLargeContent(60)
	agent.History = []api.Message{
		{Role: "user", Content: "turn 1"},
		{Role: "tool", Content: large, ToolCallID: "c1", ToolName: "search_code"},
		{Role: "user", Content: "turn 2"},
		{Role: "assistant", Content: "a2"},
		{Role: "user", Content: "turn 3"},
		{Role: "assistant", Content: "a3"},
		{Role: "user", Content: "turn 4"},
		{Role: "assistant", Content: "a4"},
		{Role: "user", Content: "turn 5"},
		{Role: "assistant", Content: "a5"},
		{Role: "user", Content: "turn 6"},
		{Role: "assistant", Content: "a6"},
		{Role: "user", Content: "turn 7"},
		{Role: "assistant", Content: "a7"},
	}

	err := agent.CompressHistory(4)
	if err != nil {
		t.Fatalf("CompressHistory() error = %v", err)
	}

	if len(provider.capturedHistory) == 0 {
		t.Fatal("ChatWithTools was not called")
	}
	capturedPrompt := provider.capturedHistory[0].Content
	if !strings.Contains(capturedPrompt, "truncated") {
		t.Error("CompressHistory() should pre-prune old tool results before BuildSummaryPrompt")
	}
}

func TestCompressHistory_UsesCompressionModelDefault(t *testing.T) {
	provider := &compressionTestProvider{name: "openai", summary: "summary"}
	cfg := config.DefaultConfig()
	cfg.Compression.Model = ""

	agent, _ := newCompressionTestAgent(t, provider, "gpt-5.4", cfg)
	agent.History = []api.Message{
		{Role: "user", Content: "message 1"},
		{Role: "assistant", Content: "message 2"},
	}

	if err := agent.CompressHistory(1); err != nil {
		t.Fatalf("CompressHistory() error = %v", err)
	}
	if provider.capturedChatModel != "gpt-5.4-mini" {
		t.Fatalf("CompressHistory() model = %q, want %q", provider.capturedChatModel, "gpt-5.4-mini")
	}
}

func TestCompressHistory_SuppressesAssistantUpdatesForSummaryRequest(t *testing.T) {
	provider := &compressionTestProvider{name: "openai", summary: "summary"}
	cfg := config.DefaultConfig()
	cfg.Output.AssistantUpdates = api.AssistantUpdatesVerbose

	agent, _ := newCompressionTestAgent(t, provider, "gpt-5.4", cfg)
	agent.History = []api.Message{
		{Role: "user", Content: "old"},
		{Role: "assistant", Content: "latest"},
	}

	if err := agent.CompressHistory(1); err != nil {
		t.Fatalf("CompressHistory() error = %v", err)
	}
	if provider.capturedChatUpdateMode != api.AssistantUpdatesOff {
		t.Fatalf("summary request assistant update mode = %q, want %q", provider.capturedChatUpdateMode, api.AssistantUpdatesOff)
	}
}
