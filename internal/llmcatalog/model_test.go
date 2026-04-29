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

func TestIsKnownModelName(t *testing.T) {
	tests := []struct {
		model string
		want  bool
	}{
		{model: "gpt-5.4", want: true},
		{model: "gpt-5.5", want: true},
		{model: "claude-sonnet-4-6", want: true},
		{model: "claude-sonnet-4.6", want: true},
		{model: "claude-sonnet-4.5", want: true},
		{model: "gemini-3.1-pro", want: true},
		{model: "amazon.nova-pro-v1:0", want: true},
		{model: "corp-gpt-5-prod", want: false},
		{model: "claude-sonnet-prod", want: false},
		{model: "unknown-model", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			if got := IsKnownModelName(tt.model); got != tt.want {
				t.Fatalf("IsKnownModelName(%q) = %v, want %v", tt.model, got, tt.want)
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

func TestKnownMaxOutputTokens_BedrockNova(t *testing.T) {
	tests := []struct {
		model string
		want  int
	}{
		{model: "amazon.nova-pro-v1:0", want: 5000},
		{model: "us.amazon.nova-pro-v1:0", want: 5000},
		{model: "eu.amazon.nova-lite-v1:0", want: 5000},
		{model: "apac.amazon.nova-micro-v1:0", want: 5000},
		{model: "amazon.nova-premier-v1:0", want: 25000},
		{model: "us.amazon.nova-premier-v1:0", want: 25000},
		{model: "global.amazon.nova-2-lite-v1:0", want: 64000},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			got, ok := KnownMaxOutputTokens(tt.model)
			if !ok {
				t.Fatalf("KnownMaxOutputTokens(%q) ok = false, want true", tt.model)
			}
			if got != tt.want {
				t.Fatalf("KnownMaxOutputTokens(%q) = %d, want %d", tt.model, got, tt.want)
			}
		})
	}
}

func TestKnownMaxOutputTokens_BedrockConverseFamilies(t *testing.T) {
	tests := []struct {
		model string
		want  int
	}{
		{model: "meta.llama3-3-70b-instruct-v1:0", want: 4000},
		{model: "us.meta.llama3-2-90b-instruct-v1:0", want: 4000},
		{model: "meta.llama4-scout-17b-instruct-v1:0", want: 8000},
		{model: "mistral.mistral-large-2402-v1:0", want: 4000},
		{model: "mistral.pixtral-large-2502-v1:0", want: 16000},
		{model: "mistral.magistral-small-2509-v1:0", want: 40000},
		{model: "mistral.ministral-14b-3-0-v1:0", want: 8000},
		{model: "cohere.command-r-v1:0", want: 4000},
		{model: "ai21.jamba-1-5-large-v1:0", want: 4000},
		{model: "writer.palmyra-x5-v1:0", want: 8000},
		{model: "deepseek.r1-v1:0", want: 8000},
		{model: "qwen.qwen3-coder-480b-a35b-instruct-v1:0", want: 16000},
		{model: "qwen.qwen3-235b-a22b-2507-v1:0", want: 8000},
		{model: "minimax.minimax-m2", want: 8000},
		{model: "nvidia.nemotron-nano-3-30b-v1:0", want: 8000},
		{model: "zai.glm-4-7-flash-v1:0", want: 4000},
		{model: "openai.gpt-oss-120b-1:0", want: 16000},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			got, ok := KnownMaxOutputTokens(tt.model)
			if !ok {
				t.Fatalf("KnownMaxOutputTokens(%q) ok = false, want true", tt.model)
			}
			if got != tt.want {
				t.Fatalf("KnownMaxOutputTokens(%q) = %d, want %d", tt.model, got, tt.want)
			}
		})
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
