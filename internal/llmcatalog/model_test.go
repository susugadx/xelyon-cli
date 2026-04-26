package llmcatalog

import "testing"

func TestKnownMaxOutputTokens_ClaudeOpus47(t *testing.T) {
	for _, model := range []string{
		"claude-opus-4-7",
		"claude-opus-4.7",
		"global.anthropic.claude-opus-4-7-v1",
		"global.anthropic.claude-opus-4-7-v1:0",
		"us.anthropic.claude-opus-4-7-v1:0",
		"anthropic/claude-opus-4-7",
	} {
		t.Run(model, func(t *testing.T) {
			got, ok := KnownMaxOutputTokens(model)
			if !ok {
				t.Fatalf("KnownMaxOutputTokens(%q) ok = false, want true", model)
			}
			if got != 128000 {
				t.Fatalf("KnownMaxOutputTokens(%q) = %d, want 128000", model, got)
			}
		})
	}
}

func TestModelContextLimit_ClaudeOpus47(t *testing.T) {
	for _, model := range []string{
		"claude-opus-4-7",
		"claude-opus-4.7",
		"global.anthropic.claude-opus-4-7-v1",
		"global.anthropic.claude-opus-4-7-v1:0",
		"us.anthropic.claude-opus-4-7-v1:0",
		"anthropic/claude-opus-4-7",
	} {
		t.Run(model, func(t *testing.T) {
			if got := ModelContextLimit(model); got != 1000000 {
				t.Fatalf("ModelContextLimit(%q) = %d, want 1000000", model, got)
			}
		})
	}
}

func TestIsAdaptiveClaudeThinkingModel_Opus47(t *testing.T) {
	tests := []struct {
		model string
		want  bool
	}{
		{"claude-opus-4-7", true},
		{"claude-opus-4.7", true},
		{"global.anthropic.claude-opus-4-7-v1:0", true},
		{" anthropic/Claude-Opus-4.7 ", true},
		{"claude-opus-4-5", false},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			if got := IsAdaptiveClaudeThinkingModel(tt.model); got != tt.want {
				t.Fatalf("IsAdaptiveClaudeThinkingModel(%q) = %v, want %v", tt.model, got, tt.want)
			}
		})
	}
}
