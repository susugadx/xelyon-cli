package agent

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/token"
)

func TestEstimateTokens(t *testing.T) {
	tests := []struct {
		name     string
		messages []api.Message
		model    string
		want     int
	}{
		{
			name:     "empty messages",
			messages: []api.Message{},
			model:    "gpt-4o",
			want:     0,
		},
		{
			name: "single short message",
			messages: []api.Message{
				{Role: "user", Content: "Hello"},
			},
			model: "gpt-4o",
			want:  token.EstimateTokenCountForModel("gpt-4o", "Hello"),
		},
		{
			name: "multiple messages",
			messages: []api.Message{
				{Role: "user", Content: "Hello"},
				{Role: "assistant", Content: "Hi!"},
				{Role: "user", Content: "How are you"},
			},
			model: "claude-sonnet-4-6",
			want: token.EstimateTokenCountForModel("claude-sonnet-4-6", "Hello") +
				token.EstimateTokenCountForModel("claude-sonnet-4-6", "Hi!") +
				token.EstimateTokenCountForModel("claude-sonnet-4-6", "How are you"),
		},
		{
			name: "long message",
			messages: []api.Message{
				{Role: "user", Content: "This is a very long message with many characters for testing purposes"},
			},
			model: "gemini-2.5-pro",
			want:  token.EstimateTokenCountForModel("gemini-2.5-pro", "This is a very long message with many characters for testing purposes"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := estimateTokens(tt.model, tt.messages)
			if got != tt.want {
				t.Errorf("estimateTokens() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAgent_CompressHistory_TooShort(t *testing.T) {
	provider := &mockProvider{name: "test"}
	agent := NewAgent("test-model", provider, false)

	agent.History = []api.Message{
		{Role: "user", Content: "msg1"},
		{Role: "assistant", Content: "msg2"},
		{Role: "user", Content: "msg3"},
	}

	err := agent.CompressHistory(5)
	if err == nil {
		t.Error("CompressHistory() should return error when history is too short")
	}
}

func TestAgent_CompressHistory_Success(t *testing.T) {
	provider := &mockProvider{name: "test"}
	agent := NewAgent("test-model", provider, false)

	for i := 0; i < 10; i++ {
		agent.History = append(agent.History, api.Message{
			Role:    "user",
			Content: "message content",
		})
	}

	initialLen := len(agent.History)

	err := agent.CompressHistory(5)
	if err != nil {
		t.Fatalf("CompressHistory() error = %v", err)
	}

	expectedLen := 6
	if len(agent.History) != expectedLen {
		t.Errorf("CompressHistory() history length = %d, want %d", len(agent.History), expectedLen)
	}
	if agent.History[0].Role != "system" {
		t.Errorf("CompressHistory() first message role = %v, want 'system'", agent.History[0].Role)
	}
	if !strings.Contains(agent.History[0].Content, "Summary") && !strings.Contains(agent.History[0].Content, "mock") {
		t.Errorf("CompressHistory() first message should be summary, got: %s", agent.History[0].Content)
	}
	if len(agent.History) >= initialLen {
		t.Errorf("CompressHistory() did not reduce history length: before=%d, after=%d", initialLen, len(agent.History))
	}
}
