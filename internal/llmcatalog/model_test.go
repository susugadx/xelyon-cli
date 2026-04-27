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

func TestKnownMaxOutputTokens_DeepSeekV4(t *testing.T) {
	for _, model := range []string{
		"deepseek-v4-flash",
		"deepseek-v4-pro",
		"deepseek-v4-custom",
		"deepseek-chat",
		"deepseek-reasoner",
	} {
		t.Run(model, func(t *testing.T) {
			got, ok := KnownMaxOutputTokens(model)
			if !ok {
				t.Fatalf("KnownMaxOutputTokens(%q) ok = false, want true", model)
			}
			if got != 384000 {
				t.Fatalf("KnownMaxOutputTokens(%q) = %d, want 384000", model, got)
			}
		})
	}
}

func TestKnownMaxOutputTokens_DeepSeekPassThrough(t *testing.T) {
	got, ok := KnownMaxOutputTokens("deepseek-coder")
	if !ok {
		t.Fatalf("KnownMaxOutputTokens(%q) ok = false, want true", "deepseek-coder")
	}
	if got != 16384 {
		t.Fatalf("KnownMaxOutputTokens(%q) = %d, want 16384", "deepseek-coder", got)
	}
}

func TestModelContextLimit_DeepSeekV4(t *testing.T) {
	for _, model := range []string{
		"deepseek-v4-flash",
		"deepseek-v4-pro",
		"deepseek-v4-custom",
		"deepseek-chat",
		"deepseek-reasoner",
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

func TestGPT55CatalogLimits(t *testing.T) {
	tests := []struct {
		model       string
		wantMaxOut  int
		wantContext int
	}{
		{model: "gpt-5.5", wantMaxOut: 128000, wantContext: 1050000},
		{model: "gpt-5.5-pro", wantMaxOut: 128000, wantContext: 1050000},
		{model: "gpt-5.5-2026-04-23", wantMaxOut: 128000, wantContext: 1050000},
		{model: "gpt-5.5-pro-2026-04-23", wantMaxOut: 128000, wantContext: 1050000},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			gotMaxOut, ok := KnownMaxOutputTokens(tt.model)
			if !ok {
				t.Fatalf("KnownMaxOutputTokens(%q) ok = false, want true", tt.model)
			}
			if gotMaxOut != tt.wantMaxOut {
				t.Fatalf("KnownMaxOutputTokens(%q) = %d, want %d", tt.model, gotMaxOut, tt.wantMaxOut)
			}
			if gotContext := ModelContextLimit(tt.model); gotContext != tt.wantContext {
				t.Fatalf("ModelContextLimit(%q) = %d, want %d", tt.model, gotContext, tt.wantContext)
			}
		})
	}
}

func TestGPT55ResponsesAPIModel(t *testing.T) {
	tests := []string{"gpt-5.5", "gpt-5.5-pro"}
	for _, model := range tests {
		t.Run(model, func(t *testing.T) {
			if !IsOpenAIResponsesModel(model, nil) {
				t.Fatalf("IsOpenAIResponsesModel(%q) = false, want true", model)
			}
		})
	}
}
