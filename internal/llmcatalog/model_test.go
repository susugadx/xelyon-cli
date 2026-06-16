package llmcatalog

import (
	"slices"
	"testing"
)

func TestKnownMaxOutputTokens_ClaudeOneMillionModels(t *testing.T) {
	for _, model := range []string{
		"claude-opus-4-8",
		"claude-opus-4.8",
		"anthropic/claude-opus-4-8",
		"anthropic/claude-opus-4.8",
		"anthropic.claude-opus-4-8",
		"global.anthropic.claude-opus-4-8",
		"us.anthropic.claude-opus-4-8",
		"eu.anthropic.claude-opus-4-8",
		"jp.anthropic.claude-opus-4-8",
		"au.anthropic.claude-opus-4-8",
		"claude-opus-4-7",
		"claude-opus-4.7",
		"claude-opus-4-6",
		"claude-opus-4.6",
		"global.anthropic.claude-opus-4-7-v1",
		"global.anthropic.claude-opus-4-7-v1:0",
		"us.anthropic.claude-opus-4-7-v1:0",
		"anthropic/claude-opus-4-7",
		"claude-fable-5",
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
		{model: "claude-opus-4-8", want: true},
		{model: "claude-opus-4.8", want: true},
		{model: "claude-fable-5", want: true},
		{model: "global.anthropic.claude-opus-4-8", want: true},
		{model: "global.anthropic.claude-sonnet-4-6", want: true},
		{model: "jp.anthropic.claude-sonnet-4-6", want: true},
		{model: "global.anthropic.claude-sonnet-4-6-v1", want: true},
		{model: "global.anthropic.claude-sonnet-4-6-v1:0", want: true},
		{model: "eu.anthropic.claude-sonnet-4-6", want: true},
		{model: "au.anthropic.claude-sonnet-4-6", want: true},
		{model: "claude-sonnet-4.5", want: true},
		{model: "gemini-3.5-flash", want: true},
		{model: "gemini-3.1-flash-lite", want: true},
		{model: "gemini-3.1-pro", want: true},
		{model: "kimi-k2.6", want: true},
		{model: "kimi-k2.5", want: true},
		{model: "kimi-k2.7-code", want: true},
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

func TestInferProviderFromModel_KnownRoutedAndUnknown(t *testing.T) {
	tests := []struct {
		model string
		want  string
	}{
		{model: " GPT-5.4 ", want: "openai"},
		{model: "o3-mini", want: "openai"},
		{model: "gemini-3.1-pro-preview", want: "gemini"},
		{model: "claude-sonnet-4-6", want: "claude"},
		{model: "claude-opus-4-8", want: "claude"},
		{model: "claude-fable-5", want: "claude"},
		{model: "jp.anthropic.claude-opus-4-8", want: "bedrock"},
		{model: "deepseek-v4-flash", want: "deepseek"},
		{model: "kimi-k2.6", want: "kimi"},
		{model: "kimi-k2.7-code", want: "kimi"},
		{model: "global.anthropic.claude-sonnet-4-6", want: "bedrock"},
		{model: "openai/gpt-5.4", want: "openrouter"},
		{model: "vendor/model", want: "openrouter"},
		{model: "unknown-model", want: ""},
		{model: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			if got := InferProviderFromModel(tt.model); got != tt.want {
				t.Fatalf("InferProviderFromModel(%q) = %q, want %q", tt.model, got, tt.want)
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

func TestKnownModelNamesForProvider_OpenAISubscriptionExactAllowlist(t *testing.T) {
	want := []string{"gpt-5.5", "gpt-5.4", "gpt-5.4-mini", "gpt-5.3-codex-spark"}
	if got := KnownModelNamesForProvider("openai_subscription"); !slices.Equal(got, want) {
		t.Fatalf("KnownModelNamesForProvider(openai_subscription) = %v, want %v", got, want)
	}
	if got := RecommendedModelNamesForProvider("chatgpt"); !slices.Equal(got, want) {
		t.Fatalf("RecommendedModelNamesForProvider(chatgpt) = %v, want %v", got, want)
	}
}

func TestRecommendedModelNamesForProvider_IncludesLatestClaudeModels(t *testing.T) {
	models := RecommendedModelNamesForProvider("claude")
	for _, model := range []string{"claude-opus-4-8", "claude-fable-5"} {
		if !slices.Contains(models, model) {
			t.Fatalf("RecommendedModelNamesForProvider(claude) = %v, want %q for /model picker", models, model)
		}
	}
}

func TestKnownAndRecommendedModelNamesForProvider_BedrockIncludesOpus48Profiles(t *testing.T) {
	models := KnownModelNamesForProvider("bedrock")
	for _, model := range []string{
		"jp.anthropic.claude-sonnet-4-6",
		"anthropic.claude-opus-4-8",
		"global.anthropic.claude-opus-4-8",
		"us.anthropic.claude-opus-4-8",
		"eu.anthropic.claude-opus-4-8",
		"jp.anthropic.claude-opus-4-8",
		"au.anthropic.claude-opus-4-8",
	} {
		if !slices.Contains(models, model) {
			t.Fatalf("KnownModelNamesForProvider(bedrock) = %v, want %q", models, model)
		}
	}
	if slices.Contains(models, "global.anthropic.claude-sonnet-4-6-v1") {
		t.Fatalf("KnownModelNamesForProvider(bedrock) should not expose duplicate Sonnet 4.6 v1 ID: %v", models)
	}
}

func TestKnownAndRecommendedModelNamesForProvider_OpenRouterExcludesFableUntilReplaySupport(t *testing.T) {
	const model = "anthropic/claude-fable-5"
	if slices.Contains(KnownModelNamesForProvider("openrouter"), model) {
		t.Fatalf("KnownModelNamesForProvider(openrouter) should not expose %q before replay support", model)
	}
	if slices.Contains(RecommendedModelNamesForProvider("openrouter"), model) {
		t.Fatalf("RecommendedModelNamesForProvider(openrouter) should not expose %q before replay support", model)
	}
}

func TestKnownAndRecommendedModelNamesForProviderDoNotDuplicateModels(t *testing.T) {
	for provider := range knownProviderModels {
		t.Run(provider+"/known", func(t *testing.T) {
			assertNoDuplicateModels(t, KnownModelNamesForProvider(provider))
		})
		t.Run(provider+"/recommended", func(t *testing.T) {
			assertNoDuplicateModels(t, RecommendedModelNamesForProvider(provider))
		})
	}
}

func assertNoDuplicateModels(t *testing.T, models []string) {
	t.Helper()
	seen := map[string]bool{}
	for _, model := range models {
		if seen[model] {
			t.Fatalf("duplicate model %q in %v", model, models)
		}
		seen[model] = true
	}
}

func TestCanonicalModelNameForProvider_GeminiResourceName(t *testing.T) {
	if got := CanonicalModelNameForProvider("gemini", "models/Gemini-3.5-Flash"); got != "gemini-3.5-flash" {
		t.Fatalf("CanonicalModelNameForProvider(gemini) = %q, want gemini-3.5-flash", got)
	}
	if got := CanonicalModelNameForProvider("openai", "models/GPT-5.4"); got != "models/GPT-5.4" {
		t.Fatalf("CanonicalModelNameForProvider(openai) = %q, want provider-specific form preserved", got)
	}
}

func TestIsKnownModelNameForProvider_GeminiResourceName(t *testing.T) {
	if !IsKnownModelNameForProvider("gemini", "models/gemini-3.5-flash") {
		t.Fatal("IsKnownModelNameForProvider(gemini, models/gemini-3.5-flash) = false, want true")
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

func TestRecommendedModelNamesForProvider_IncludesRecommendedGeminiModels(t *testing.T) {
	models := RecommendedModelNamesForProvider("gemini")
	wantPrefix := []string{
		"gemini-3.5-flash",
		"gemini-3.1-flash-lite",
		"gemini-3.1-pro-preview-customtools",
	}
	if len(models) < len(wantPrefix) {
		t.Fatalf("RecommendedModelNamesForProvider(gemini) = %v, want at least %d models", models, len(wantPrefix))
	}
	if !slices.Equal(models[:len(wantPrefix)], wantPrefix) {
		t.Fatalf("RecommendedModelNamesForProvider(gemini) prefix = %v, want %v; all=%v", models[:len(wantPrefix)], wantPrefix, models)
	}

	hiddenModels := []string{
		"gemini-3.1-pro",
		"gemini-3.1-pro-preview",
		"gemini-3-pro-preview",
		"gemini-2.0-flash",
		"gemini-2.0-flash-001",
		"gemini-2.0-flash-exp",
		"gemini-2.0-flash-lite",
		"gemini-2.0-flash-lite-001",
		"gemini-1.5-pro",
		"gemini-1.5-flash",
		"gemini-3.1-flash-lite-preview",
	}
	for _, model := range hiddenModels {
		if slices.Contains(models, model) {
			t.Fatalf("RecommendedModelNamesForProvider(gemini) = %v, should not expose %q", models, model)
		}
	}
}

func TestKnownModelNamesForProvider_IncludesHiddenGeminiCatalogModels(t *testing.T) {
	models := KnownModelNamesForProvider("gemini")
	for _, model := range []string{
		"gemini-3.1-pro-preview",
		"gemini-3.1-pro",
		"gemini-2.0-flash-exp",
	} {
		if !slices.Contains(models, model) {
			t.Fatalf("KnownModelNamesForProvider(gemini) = %v, want hidden catalog model %q", models, model)
		}
	}
}

func TestModelLifecycleForProvider_GeminiPickerAndShutdownMetadata(t *testing.T) {
	tests := []struct {
		model        string
		stage        ModelLifecycleStage
		hidden       bool
		shutdownDate string
		replacement  string
		warn         bool
	}{
		{
			model: "gemini-3.5-flash",
			stage: ModelLifecycleActive,
		},
		{
			model:       "gemini-3.1-pro",
			stage:       ModelLifecycleActive,
			hidden:      true,
			replacement: "gemini-3.1-pro-preview-customtools",
			warn:        true,
		},
		{
			model:        "gemini-3-pro-preview",
			stage:        ModelLifecycleShutdown,
			hidden:       true,
			shutdownDate: "2026-03-09",
			replacement:  "gemini-3.1-pro-preview-customtools",
			warn:         true,
		},
		{
			model:        "gemini-2.0-flash",
			stage:        ModelLifecycleDeprecated,
			hidden:       true,
			shutdownDate: "2026-06-01",
			replacement:  "gemini-3.5-flash",
			warn:         true,
		},
		{
			model:        "gemini-1.5-pro-001",
			stage:        ModelLifecycleShutdown,
			hidden:       true,
			shutdownDate: "2025-09-29",
			replacement:  "gemini-3.1-pro-preview-customtools",
			warn:         true,
		},
		{
			model:        "gemini-1.5-flash-latest",
			stage:        ModelLifecycleShutdown,
			hidden:       true,
			shutdownDate: "2025-09-29",
			replacement:  "gemini-3.5-flash",
			warn:         true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			got, ok := ModelLifecycleForProvider("gemini", tt.model)
			if !ok {
				t.Fatalf("ModelLifecycleForProvider(gemini, %q) ok = false, want true", tt.model)
			}
			if got.Stage != tt.stage ||
				got.HiddenFromPicker != tt.hidden ||
				got.ShutdownDate != tt.shutdownDate ||
				got.Replacement != tt.replacement ||
				got.ShouldWarn() != tt.warn {
				t.Fatalf("ModelLifecycleForProvider(gemini, %q) = %#v, want stage=%s hidden=%t shutdown=%q replacement=%q warn=%t", tt.model, got, tt.stage, tt.hidden, tt.shutdownDate, tt.replacement, tt.warn)
			}
		})
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
		{provider: "openai_subscription", model: "gpt-5.5", want: true},
		{provider: "openai_subscription", model: "gpt-5.3-codex-spark", want: true},
		{provider: "openai_subscription", model: "gpt-5.3-codex", want: false},
		{provider: "openai_subscription", model: "gpt-5.2", want: false},
		{provider: "openai", model: "meta-llama/llama-4-scout-17b-16e-instruct", want: false},
		{provider: "openrouter", model: "anthropic/claude-sonnet-4.6", want: true},
		{provider: "openrouter", model: "anthropic/claude-fable-5", want: false},
		{provider: "gemini", model: "gemini-3.1-pro", want: true},
		{provider: "gemini", model: "gemini-3-pro-preview", want: true},
		{provider: "gemini", model: "gemini-2.0-flash", want: true},
		{provider: "azure", model: "gpt-5.4", want: false},
		{provider: "bedrock", model: "global.anthropic.claude-opus-4-8", want: true},
		{provider: "bedrock", model: "jp.anthropic.claude-opus-4-8", want: true},
		{provider: "bedrock", model: "global.anthropic.claude-sonnet-4-6-v1", want: true},
		{provider: "bedrock", model: "jp.anthropic.claude-sonnet-4-6-v1:0", want: true},
		{provider: "bedrock", model: "meta.llama3-3-70b-instruct-v1:0", want: true},
		{provider: "bedrock", model: "gpt-5.4", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.provider+"/"+tt.model, func(t *testing.T) {
			if got := IsKnownModelNameForProvider(tt.provider, tt.model); got != tt.want {
				t.Fatalf("IsKnownModelNameForProvider(%q, %q) = %v, want %v", tt.provider, tt.model, got, tt.want)
			}
		})
	}
}

func TestModelContextLimit_ClaudeOneMillionModels(t *testing.T) {
	for _, model := range []string{
		"claude-opus-4-8",
		"claude-opus-4.8",
		"anthropic/claude-opus-4-8",
		"anthropic/claude-opus-4.8",
		"anthropic.claude-opus-4-8",
		"global.anthropic.claude-opus-4-8",
		"us.anthropic.claude-opus-4-8",
		"eu.anthropic.claude-opus-4-8",
		"jp.anthropic.claude-opus-4-8",
		"au.anthropic.claude-opus-4-8",
		"claude-opus-4-7",
		"claude-opus-4.7",
		"global.anthropic.claude-opus-4-7-v1",
		"global.anthropic.claude-opus-4-7-v1:0",
		"us.anthropic.claude-opus-4-7-v1:0",
		"anthropic/claude-opus-4-7",
		"claude-fable-5",
		"claude-sonnet-4-6",
		"claude-sonnet-4.6",
		"anthropic/claude-sonnet-4-6",
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
		"jp.anthropic.claude-sonnet-4-6",
		"au.anthropic.claude-sonnet-4-6",
		"anthropic.claude-sonnet-4-6-v1",
		"global.anthropic.claude-sonnet-4-6-v1",
		"jp.anthropic.claude-sonnet-4-6-v1:0",
	} {
		t.Run(model, func(t *testing.T) {
			if got := ModelContextLimit(model); got != 1000000 {
				t.Fatalf("ModelContextLimit(%q) = %d, want 1000000", model, got)
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
		{model: "claude-sonnet-4-6", want: 1000000, ok: true},
		{model: "claude-opus-4-8", want: 1000000, ok: true},
		{model: "global.anthropic.claude-opus-4-8", want: 1000000, ok: true},
		{model: "claude-opus-4-6", want: 1000000, ok: true},
		{model: "claude-fable-5", want: 1000000, ok: true},
		{model: "jp.anthropic.claude-sonnet-4-6", want: 1000000, ok: true},
		{model: "global.anthropic.claude-sonnet-4-6-v1", want: 1000000, ok: true},
		{model: "global.anthropic.claude-sonnet-4-6-v1:0", want: 1000000, ok: true},
		{model: "deepseek-v4-custom", want: 1000000, ok: true},
		{model: "kimi-k2.6", want: 256000, ok: true},
		{model: "kimi-k2.5", want: 256000, ok: true},
		{model: "kimi-k2.7-code", want: 256000, ok: true},
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
		{model: "anthropic/claude-opus-4.8", wantContext: 1000000, wantMaxOutput: 128000},
		{model: "anthropic/claude-opus-4.6", wantContext: 1000000, wantMaxOutput: 128000},
		{model: "anthropic/claude-sonnet-4.6", wantContext: 1000000, wantMaxOutput: 64000},
		{model: "anthropic/claude-sonnet-4-6", wantContext: 1000000, wantMaxOutput: 64000},
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

func TestKnownModelLimits_OpenRouterFableUnsupportedUntilReplaySupport(t *testing.T) {
	const model = "anthropic/claude-fable-5"
	if got, ok := KnownModelContextLimit(model); ok || got != 0 {
		t.Fatalf("KnownModelContextLimit(%q) = %d, %v; want 0, false", model, got, ok)
	}
	if got, ok := KnownMaxOutputTokens(model); ok || got != 0 {
		t.Fatalf("KnownMaxOutputTokens(%q) = %d, %v; want 0, false", model, got, ok)
	}
	if IsKnownModelName(model) {
		t.Fatalf("IsKnownModelName(%q) = true, want false", model)
	}
}

func TestKnownModelLimits_BedrockSonnet46VersionedSupported(t *testing.T) {
	for _, model := range []string{
		"anthropic.claude-sonnet-4-6-v1",
		"global.anthropic.claude-sonnet-4-6-v1",
		"global.anthropic.claude-sonnet-4-6-v1:0",
		"jp.anthropic.claude-sonnet-4-6-v1:0",
	} {
		t.Run(model, func(t *testing.T) {
			if got, ok := KnownModelContextLimit(model); !ok || got != 1000000 {
				t.Fatalf("KnownModelContextLimit(%q) = %d, %v; want 1000000, true", model, got, ok)
			}
			if got, ok := KnownMaxOutputTokens(model); !ok || got != 64000 {
				t.Fatalf("KnownMaxOutputTokens(%q) = %d, %v; want 64000, true", model, got, ok)
			}
			if !IsKnownModelName(model) {
				t.Fatalf("IsKnownModelName(%q) = false, want true", model)
			}
			if !IsKnownModelNameForProvider("bedrock", model) {
				t.Fatalf("IsKnownModelNameForProvider(bedrock, %q) = false, want true", model)
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
		"kimi-k2.7-code",
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

func TestKimiBuiltinWebSearchRequestModel(t *testing.T) {
	tests := []struct {
		name         string
		model        string
		catalogModel string
		wantModel    string
		wantAdjusted bool
	}{
		{name: "k2.7 request model falls back to k2.6", model: "kimi-k2.7-code", wantModel: "kimi-k2.6", wantAdjusted: true},
		{name: "k2.7 catalog model falls back to k2.6", model: "corp-kimi", catalogModel: "kimi-k2.7-code", wantModel: "kimi-k2.6", wantAdjusted: true},
		{name: "k2.6 stays unchanged", model: "kimi-k2.6", wantModel: "kimi-k2.6"},
		{name: "empty model stays empty", wantModel: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotModel, gotAdjusted := KimiBuiltinWebSearchRequestModel(tt.model, tt.catalogModel)
			if gotModel != tt.wantModel || gotAdjusted != tt.wantAdjusted {
				t.Fatalf("KimiBuiltinWebSearchRequestModel(%q, %q) = %q, %t; want %q, %t",
					tt.model, tt.catalogModel, gotModel, gotAdjusted, tt.wantModel, tt.wantAdjusted)
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
		{model: "anthropic.claude-opus-4-8", want: 128000},
		{model: "global.anthropic.claude-opus-4-8", want: 128000},
		{model: "us.anthropic.claude-opus-4-8", want: 128000},
		{model: "eu.anthropic.claude-opus-4-8", want: 128000},
		{model: "jp.anthropic.claude-opus-4-8", want: 128000},
		{model: "au.anthropic.claude-opus-4-8", want: 128000},
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
		{"global.anthropic.claude-opus-4-8", true},
		{"jp.anthropic.claude-opus-4-8", true},
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
