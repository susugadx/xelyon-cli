package llmcatalog

import (
	"slices"
	"testing"
)

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

func TestKnownMaxOutputTokens_GeminiFlashModels(t *testing.T) {
	for _, model := range []string{
		"gemini-3.5-flash",
		"gemini-3.1-flash-lite",
	} {
		t.Run(model, func(t *testing.T) {
			got, ok := KnownMaxOutputTokens(model)
			if !ok {
				t.Fatalf("KnownMaxOutputTokens(%q) ok = false, want true", model)
			}
			if got != 65536 {
				t.Fatalf("KnownMaxOutputTokens(%q) = %d, want 65536", model, got)
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
		{model: "global.anthropic.claude-sonnet-4-6", want: true},
		{model: "eu.anthropic.claude-sonnet-4-6", want: true},
		{model: "au.anthropic.claude-sonnet-4-6", want: true},
		{model: "claude-sonnet-4.5", want: true},
		{model: "gemini-3.5-flash", want: true},
		{model: "gemini-3.1-flash-lite", want: true},
		{model: "gemini-3.1-pro", want: true},
		{model: "kimi-k2.6", want: true},
		{model: "kimi-k2.5", want: true},
		{model: "kimi-k2-thinking", want: true},
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

func TestKnownModelNamesForProvider_ReturnsStableClonedProviderModels(t *testing.T) {
	models := KnownModelNamesForProvider("openai")
	if len(models) < 3 {
		t.Fatalf("KnownModelNamesForProvider(openai) len = %d, want known models", len(models))
	}
	wantPrefix := []string{"gpt-5.5", "gpt-5.5-pro", "gpt-5.4"}
	for i, want := range wantPrefix {
		if models[i] != want {
			t.Fatalf("KnownModelNamesForProvider(openai)[%d] = %q, want %q; all=%v", i, models[i], want, models)
		}
	}

	models[0] = "mutated"
	again := KnownModelNamesForProvider("openai")
	if again[0] != "gpt-5.5" {
		t.Fatalf("KnownModelNamesForProvider should return a clone, got %q", again[0])
	}
}

func TestKnownModelNamesForProvider_AzureDoesNotUseCatalogModels(t *testing.T) {
	if got := KnownModelNamesForProvider("azure"); len(got) != 0 {
		t.Fatalf("KnownModelNamesForProvider(azure) = %v, want no deployment catalog", got)
	}
}

func TestKnownModelNamesForProvider_IncludesGPT53CodexForOpenAIProviders(t *testing.T) {
	tests := []struct {
		provider string
		model    string
	}{
		{provider: "openai", model: "gpt-5.3-codex"},
		{provider: "openrouter", model: "openai/gpt-5.3-codex"},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			models := KnownModelNamesForProvider(tt.provider)
			if !slices.Contains(models, tt.model) {
				t.Fatalf("KnownModelNamesForProvider(%q) = %v, want %q", tt.provider, models, tt.model)
			}
		})
	}
}

func TestKnownModelNamesForProvider_IncludesRecommendedGeminiModels(t *testing.T) {
	models := KnownModelNamesForProvider("gemini")
	wantPrefix := []string{
		"gemini-3.5-flash",
		"gemini-3.1-flash-lite",
		"gemini-3.1-pro-preview-customtools",
	}
	if len(models) < len(wantPrefix) {
		t.Fatalf("KnownModelNamesForProvider(gemini) = %v, want at least %d models", models, len(wantPrefix))
	}
	if !slices.Equal(models[:len(wantPrefix)], wantPrefix) {
		t.Fatalf("KnownModelNamesForProvider(gemini) prefix = %v, want %v; all=%v", models[:len(wantPrefix)], wantPrefix, models)
	}

	hiddenModels := []string{
		"gemini-3.1-pro",
		"gemini-3.1-pro-preview",
		"gemini-3-pro-preview",
		"gemini-2.0-flash",
		"gemini-2.0-flash-exp",
		"gemini-1.5-pro",
		"gemini-1.5-flash",
	}
	for _, model := range hiddenModels {
		if slices.Contains(models, model) {
			t.Fatalf("KnownModelNamesForProvider(gemini) = %v, should not expose %q", models, model)
		}
	}
}

func TestIsKnownModelNameForProvider_UsesProviderScopedCatalog(t *testing.T) {
	tests := []struct {
		provider string
		model    string
		want     bool
	}{
		{provider: "groq", model: "meta-llama/llama-4-scout-17b-16e-instruct", want: true},
		{provider: "groq", model: "llama-3.2-11b-vision-preview", want: true},
		{provider: "groq", model: "gpt-5.4", want: false},
		{provider: "deepseek", model: "deepseek-v4-custom", want: true},
		{provider: "deepseek", model: "gpt-5.4", want: false},
		{provider: "kimi", model: "kimi-k2.6", want: true},
		{provider: "kimi", model: "kimi-k2-custom", want: true},
		{provider: "kimi", model: "moonshotai.kimi-k2.5", want: false},
		{provider: "openai", model: "gpt-5.4", want: true},
		{provider: "openai", model: "meta-llama/llama-4-scout-17b-16e-instruct", want: false},
		{provider: "azure", model: "gpt-5.4", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.provider+"/"+tt.model, func(t *testing.T) {
			if got := IsKnownModelNameForProvider(tt.provider, tt.model); got != tt.want {
				t.Fatalf("IsKnownModelNameForProvider(%q, %q) = %v, want %v", tt.provider, tt.model, got, tt.want)
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

func TestModelContextLimit_BedrockClaudeProfiles(t *testing.T) {
	for _, model := range []string{
		"anthropic.claude-sonnet-4-6",
		"global.anthropic.claude-sonnet-4-6",
		"us.anthropic.claude-sonnet-4-6",
		"eu.anthropic.claude-sonnet-4-6",
		"au.anthropic.claude-sonnet-4-6",
	} {
		t.Run(model, func(t *testing.T) {
			if got := ModelContextLimit(model); got != 200000 {
				t.Fatalf("ModelContextLimit(%q) = %d, want 200000", model, got)
			}
		})
	}
}

func TestKnownModelContextLimit(t *testing.T) {
	tests := []struct {
		model string
		want  int
		ok    bool
	}{
		{model: "gpt-5.4", want: 1000000, ok: true},
		{model: "gemini-3.5-flash", want: 1048576, ok: true},
		{model: "gemini-3.1-flash-lite", want: 1000000, ok: true},
		{model: "claude-sonnet-4-6", want: 200000, ok: true},
		{model: "deepseek-v4-custom", want: 1000000, ok: true},
		{model: "kimi-k2.6", want: 256000, ok: true},
		{model: "kimi-k2.5", want: 256000, ok: true},
		{model: "kimi-k2-thinking", want: 256000, ok: true},
		{model: "corp-gpt-deployment", ok: false},
		{model: "", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			got, ok := KnownModelContextLimit(tt.model)
			if ok != tt.ok {
				t.Fatalf("KnownModelContextLimit(%q) ok = %v, want %v", tt.model, ok, tt.ok)
			}
			if got != tt.want {
				t.Fatalf("KnownModelContextLimit(%q) = %d, want %d", tt.model, got, tt.want)
			}
		})
	}
}

func TestKnownModelLimits_OpenRouterDelegatedModels(t *testing.T) {
	tests := []struct {
		model         string
		wantContext   int
		wantMaxOutput int
	}{
		{model: "anthropic/claude-sonnet-4.6", wantContext: 200000, wantMaxOutput: 64000},
		{model: "anthropic/claude-sonnet-4-6", wantContext: 200000, wantMaxOutput: 64000},
		{model: "google/gemini-3.1-pro", wantContext: 1000000, wantMaxOutput: 65536},
		{model: "deepseek/deepseek-v4-flash", wantContext: 1000000, wantMaxOutput: 384000},
		{model: "moonshotai/kimi-k2.6", wantContext: 256000, wantMaxOutput: 32768},
		{model: "moonshotai/kimi-k2-thinking", wantContext: 256000, wantMaxOutput: 32768},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			gotContext, ok := KnownModelContextLimit(tt.model)
			if !ok {
				t.Fatalf("KnownModelContextLimit(%q) ok = false, want true", tt.model)
			}
			if gotContext != tt.wantContext {
				t.Fatalf("KnownModelContextLimit(%q) = %d, want %d", tt.model, gotContext, tt.wantContext)
			}

			gotOutput, ok := KnownMaxOutputTokens(tt.model)
			if !ok {
				t.Fatalf("KnownMaxOutputTokens(%q) ok = false, want true", tt.model)
			}
			if gotOutput != tt.wantMaxOutput {
				t.Fatalf("KnownMaxOutputTokens(%q) = %d, want %d", tt.model, gotOutput, tt.wantMaxOutput)
			}
		})
	}
}

func TestInferProviderFromModel_KimiThinkingCompatibility(t *testing.T) {
	tests := []struct {
		model string
		want  string
	}{
		{model: "kimi-k2-thinking", want: "kimi"},
		{model: "moonshotai/kimi-k2-thinking", want: "openrouter"},
		{model: "moonshotai.kimi-k2-thinking", want: "bedrock"},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			if got := InferProviderFromModel(tt.model); got != tt.want {
				t.Fatalf("InferProviderFromModel(%q) = %q, want %q", tt.model, got, tt.want)
			}
		})
	}
}

func TestKnownMaxOutputTokens_Kimi(t *testing.T) {
	for _, model := range []string{
		"kimi-k2.6",
		"kimi-k2.5",
		"kimi-k2",
		"kimi-k2-thinking",
	} {
		t.Run(model, func(t *testing.T) {
			got, ok := KnownMaxOutputTokens(model)
			if !ok {
				t.Fatalf("KnownMaxOutputTokens(%q) ok = false, want true", model)
			}
			if got != 32768 {
				t.Fatalf("KnownMaxOutputTokens(%q) = %d, want 32768", model, got)
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

func TestKnownMaxOutputTokens_BedrockClaudeProfiles(t *testing.T) {
	tests := []struct {
		model string
		want  int
	}{
		{model: "anthropic.claude-sonnet-4-6", want: 64000},
		{model: "global.anthropic.claude-sonnet-4-6", want: 64000},
		{model: "us.anthropic.claude-sonnet-4-6", want: 64000},
		{model: "eu.anthropic.claude-sonnet-4-6", want: 64000},
		{model: "au.anthropic.claude-sonnet-4-6", want: 64000},
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
		{model: "us.mistral.pixtral-large-2502-v1:0", want: 16000},
		{model: "mistral.magistral-small-2509-v1:0", want: 40000},
		{model: "mistral.ministral-14b-3-0-v1:0", want: 8000},
		{model: "cohere.command-r-v1:0", want: 4000},
		{model: "ai21.jamba-1-5-large-v1:0", want: 4000},
		{model: "writer.palmyra-x5-v1:0", want: 8000},
		{model: "us.writer.palmyra-x5-v1:0", want: 8000},
		{model: "deepseek.r1-v1:0", want: 8000},
		{model: "us.deepseek.r1-v1:0", want: 8000},
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

func TestGPT53CodexCatalogLimits(t *testing.T) {
	const model = "gpt-5.3-codex"

	gotMaxOut, ok := KnownMaxOutputTokens(model)
	if !ok {
		t.Fatalf("KnownMaxOutputTokens(%q) ok = false, want true", model)
	}
	if gotMaxOut != 128000 {
		t.Fatalf("KnownMaxOutputTokens(%q) = %d, want 128000", model, gotMaxOut)
	}
	if gotContext := ModelContextLimit(model); gotContext != 400000 {
		t.Fatalf("ModelContextLimit(%q) = %d, want 400000", model, gotContext)
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
