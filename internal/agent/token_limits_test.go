package agent

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/agent/token"
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
		{"gpt-5.1", 400000},
		{"deepseek-chat", 1000000},
		{"gemini-2.0-flash", 1000000},

		// Prefix matches
		{"claude-3-opus-latest", 200000},
		{"gpt-4o-2024-01-01", 128000},
		{"gpt-5-preview", 400000},
		{"deepseek-coder-v3", 64000},
		{"deepseek-v4-custom", 1000000},
		{"deepseek-v3", 128000},
		{"gemini-2.5-pro-exp", 1000000},

		// Default fallback
		{"unknown-model-xyz", 100000},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			got := token.GetModelTokenLimit(tt.model)
			if got != tt.expected {
				t.Errorf("token.GetModelTokenLimit(%q) = %d, want %d", tt.model, got, tt.expected)
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
			got := token.EstimateTokenCount(tt.text)
			if got < tt.minToken || got > tt.maxToken {
				t.Errorf("token.EstimateTokenCount(%q) = %d, want between %d and %d",
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

func TestAgent_EstimateTokens_UsesProviderFacingHistoryReduction(t *testing.T) {
	callID := "call_budget_old"
	oldRead := strings.Repeat("あ", 12000)
	agent := &Agent{
		CurrentModel: "gpt-5.4",
		SystemPrompt: "system",
		Runtime: &AgentRuntime{
			Options: RuntimeOptions{EnableProviderHistoryReduction: true},
			TaskLedger: providerHistoryTaskLedgerWithEvidence(t,
				providerHistoryEvidenceItem{ToolName: "read_file", ToolCallID: callID, Path: "README.md", StartLine: 1, EndLine: 3},
			),
			LastProviderHistoryProjectionReport: ProviderHistoryProjectionReport{
				Mode:           ProviderHistoryReductionDryRun,
				CandidateCount: 42,
			},
		},
		History: providerHistoryReductionRequestHistory(callID, oldRead),
	}
	wantReport := agent.Runtime.LastProviderHistoryProjectionReport

	projected := agent.buildProviderHistoryProjection(ProviderHistoryReductionPolicy{Mode: ProviderHistoryReductionApply}).History
	wantHistoryTokens := estimateTokens(agent.CurrentModel, projected)
	if rawHistoryTokens := estimateTokens(agent.CurrentModel, agent.History); wantHistoryTokens >= rawHistoryTokens {
		t.Fatalf("projected history tokens = %d, want less than raw history tokens %d", wantHistoryTokens, rawHistoryTokens)
	}

	if got := agent.EstimateHistoryTokens(); got != wantHistoryTokens {
		t.Fatalf("EstimateHistoryTokens() = %d, want provider-facing history tokens %d", got, wantHistoryTokens)
	}
	wantTotal := token.EstimateTokenCountForModel(agent.CurrentModel, agent.SystemPrompt) + wantHistoryTokens
	if got := agent.EstimateTokens(); got != wantTotal {
		t.Fatalf("EstimateTokens() = %d, want system + provider-facing history = %d", got, wantTotal)
	}
	assertLastProviderHistoryProjectionReportPreserved(t, agent.Runtime, wantReport)
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
