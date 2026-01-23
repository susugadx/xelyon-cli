package agent

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
)

func TestGetModelTokenLimit(t *testing.T) {
	tests := []struct {
		model    string
		expected int
	}{
		// Exact matches
		{"claude-sonnet-4-20250514", 200000},
		{"gpt-4o", 128000},
		{"deepseek-chat", 64000},
		{"gemini-2.0-flash", 1000000},

		// Prefix matches
		{"claude-3-opus-latest", 200000},
		{"gpt-4o-2024-01-01", 128000},
		{"deepseek-coder-v3", 64000},
		{"gemini-2.5-pro-exp", 1000000},

		// Default fallback
		{"unknown-model-xyz", 100000},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			got := GetModelTokenLimit(tt.model)
			if got != tt.expected {
				t.Errorf("GetModelTokenLimit(%q) = %d, want %d", tt.model, got, tt.expected)
			}
		})
	}
}

func TestEstimateTokenCount(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		minToken int
		maxToken int
	}{
		{"empty", "", 0, 0},
		{"short_english", "hello", 1, 5},
		{"medium_english", "hello world", 2, 10},
		{"japanese", "こんにちは", 2, 10},
		{"mixed", "Hello こんにちは World", 5, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := estimateTokenCount(tt.text)
			if got < tt.minToken || got > tt.maxToken {
				t.Errorf("estimateTokenCount(%q) = %d, want between %d and %d",
					tt.text, got, tt.minToken, tt.maxToken)
			}
		})
	}
}

func TestAgent_EstimateTokens(t *testing.T) {
	agent := &Agent{
		SystemPrompt: "You are a helpful assistant.",
		History: []api.Message{
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "Hi there! How can I help you?"},
		},
	}

	tokens := agent.EstimateTokens()
	if tokens <= 0 {
		t.Errorf("EstimateTokens() = %d, want > 0", tokens)
	}
}

func TestAgent_EstimateSystemPromptTokens(t *testing.T) {
	agent := &Agent{
		SystemPrompt: "You are a helpful assistant.",
	}

	tokens := agent.EstimateSystemPromptTokens()
	if tokens <= 0 {
		t.Errorf("EstimateSystemPromptTokens() = %d, want > 0", tokens)
	}
}

func TestAgent_EstimateHistoryTokens(t *testing.T) {
	agent := &Agent{
		History: []api.Message{
			{Role: "user", Content: "Hello"},
			{Role: "assistant", Content: "Hi there!"},
		},
	}

	tokens := agent.EstimateHistoryTokens()
	if tokens <= 0 {
		t.Errorf("EstimateHistoryTokens() = %d, want > 0", tokens)
	}
}

func TestAgent_EstimateHistoryTokens_Empty(t *testing.T) {
	agent := &Agent{
		History: []api.Message{},
	}

	tokens := agent.EstimateHistoryTokens()
	if tokens != 0 {
		t.Errorf("EstimateHistoryTokens() = %d, want 0 for empty history", tokens)
	}
}

func TestAgent_GetTokenUsagePercentage(t *testing.T) {
	agent := &Agent{
		SystemPrompt: "You are a helpful assistant.",
		CurrentModel: "claude-sonnet-4-20250514",
		History: []api.Message{
			{Role: "user", Content: "Hello"},
		},
	}

	percentage := agent.GetTokenUsagePercentage()
	if percentage <= 0 || percentage >= 100 {
		t.Errorf("GetTokenUsagePercentage() = %f, want between 0 and 100", percentage)
	}
}

func TestAgent_GetTokenUsagePercentage_ZeroLimit(t *testing.T) {
	agent := &Agent{
		SystemPrompt: "Test",
		CurrentModel: "", // Will use default
		History:      []api.Message{},
	}

	// With unknown model, should use default limit and return a small percentage
	percentage := agent.GetTokenUsagePercentage()
	if percentage < 0 {
		t.Errorf("GetTokenUsagePercentage() = %f, should not be negative", percentage)
	}
}
