package config

import "testing"

func TestGetSelectedModelForProvider_Matrix(t *testing.T) {
	openAIDefault := DefaultConfig().ProviderModels["openai"].DefaultModel
	claudeDefault := DefaultConfig().ProviderModels["claude"].DefaultModel
	ollamaDefault := DefaultConfig().ProviderModels["ollama"].DefaultModel

	tests := []struct {
		name            string
		defaultProvider string
		defaultModel    string
		state           providerModelSectionState
		raw             map[string]ProviderModelConfig
		provider        string
		want            string
	}{
		{
			name:            "absent uses global default for default provider",
			defaultProvider: "openai",
			defaultModel:    "gpt-global",
			state:           providerModelSectionStateAbsent,
			provider:        "openai",
			want:            "gpt-global",
		},
		{
			name:            "absent ignores known model from different provider",
			defaultProvider: "openai",
			defaultModel:    "deepseek-chat",
			state:           providerModelSectionStateAbsent,
			provider:        "openai",
			want:            openAIDefault,
		},
		{
			name:            "absent uses provider default for non default provider",
			defaultProvider: "deepseek",
			defaultModel:    "deepseek-global",
			state:           providerModelSectionStateAbsent,
			provider:        "openai",
			want:            openAIDefault,
		},
		{
			name:            "absent keeps ollama local model even if it looks foreign",
			defaultProvider: "ollama",
			defaultModel:    "deepseek-r1:8b",
			state:           providerModelSectionStateAbsent,
			provider:        "ollama",
			want:            "deepseek-r1:8b",
		},
		{
			name:            "explicit empty keeps ollama arbitrary model names",
			defaultProvider: "ollama",
			defaultModel:    "gpt-oss:20b",
			state:           providerModelSectionStateExplicitEmpty,
			provider:        "ollama",
			want:            "gpt-oss:20b",
		},
		{
			name:            "explicit empty keeps alias default provider fallback",
			defaultProvider: "claude",
			defaultModel:    "claude-global",
			state:           providerModelSectionStateExplicitEmpty,
			provider:        "anthropic",
			want:            "claude-global",
		},
		{
			name:            "explicit empty does not leak global model to other provider",
			defaultProvider: "deepseek",
			defaultModel:    "deepseek-global",
			state:           providerModelSectionStateExplicitEmpty,
			provider:        "anthropic",
			want:            claudeDefault,
		},
		{
			name:            "explicit entry shadows top level default",
			defaultProvider: "openai",
			defaultModel:    "gpt-global",
			state:           providerModelSectionStateImplicitEntries,
			raw: map[string]ProviderModelConfig{
				"openai": {DefaultModel: "gpt-explicit"},
			},
			provider: "openai",
			want:     "gpt-explicit",
		},
		{
			name:            "partial explicit entries keep default provider global model",
			defaultProvider: "openai",
			defaultModel:    "gpt-global",
			state:           providerModelSectionStateImplicitEntries,
			raw: map[string]ProviderModelConfig{
				"deepseek": {DefaultModel: "deepseek-explicit"},
			},
			provider: "openai",
			want:     "gpt-global",
		},
		{
			name:            "partial explicit entries ignore known model from different provider for default provider",
			defaultProvider: "openai",
			defaultModel:    "deepseek-chat",
			state:           providerModelSectionStateImplicitEntries,
			raw: map[string]ProviderModelConfig{
				"openai": {MaxOutputTokens: 999},
			},
			provider: "openai",
			want:     openAIDefault,
		},
		{
			name:            "partial explicit entries ignore custom model already owned by different provider",
			defaultProvider: "openai",
			defaultModel:    "custom-shared",
			state:           providerModelSectionStateImplicitEntries,
			raw: map[string]ProviderModelConfig{
				"claude": {DefaultModel: "custom-shared"},
			},
			provider: "openai",
			want:     openAIDefault,
		},
		{
			name:            "partial explicit entries keep provider default for unrelated provider",
			defaultProvider: "openai",
			defaultModel:    "gpt-global",
			state:           providerModelSectionStateImplicitEntries,
			raw: map[string]ProviderModelConfig{
				"deepseek": {DefaultModel: "deepseek-explicit"},
			},
			provider: "claude",
			want:     claudeDefault,
		},
		{
			name:            "partial explicit entries still keep ollama custom model",
			defaultProvider: "ollama",
			defaultModel:    "deepseek-r1:8b",
			state:           providerModelSectionStateImplicitEntries,
			raw: map[string]ProviderModelConfig{
				"openai": {DefaultModel: "gpt-explicit"},
			},
			provider: "ollama",
			want:     "deepseek-r1:8b",
		},
		{
			name:            "partial explicit entries keep ollama provider default for unrelated provider",
			defaultProvider: "ollama",
			defaultModel:    "gpt-oss:20b",
			state:           providerModelSectionStateImplicitEntries,
			raw: map[string]ProviderModelConfig{
				"deepseek": {DefaultModel: "deepseek-explicit"},
			},
			provider: "openai",
			want:     openAIDefault,
		},
		{
			name:            "explicit empty still keeps ollama built-in default for unrelated provider",
			defaultProvider: "ollama",
			defaultModel:    "deepseek-r1:8b",
			state:           providerModelSectionStateExplicitEmpty,
			provider:        "claude",
			want:            claudeDefault,
		},
		{
			name:            "absent still exposes built-in ollama default when no global override",
			defaultProvider: "ollama",
			defaultModel:    "",
			state:           providerModelSectionStateAbsent,
			provider:        "ollama",
			want:            ollamaDefault,
		},
		{
			name:            "absent keeps groq slash model",
			defaultProvider: "groq",
			defaultModel:    "moonshotai/kimi-k2-instruct",
			state:           providerModelSectionStateAbsent,
			provider:        "groq",
			want:            "moonshotai/kimi-k2-instruct",
		},
		{
			name:            "partial explicit entries still keep groq slash model",
			defaultProvider: "groq",
			defaultModel:    "moonshotai/kimi-k2-instruct",
			state:           providerModelSectionStateImplicitEntries,
			raw: map[string]ProviderModelConfig{
				"openai": {DefaultModel: "gpt-explicit"},
			},
			provider: "groq",
			want:     "moonshotai/kimi-k2-instruct",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			cfg.DefaultProvider = tt.defaultProvider
			cfg.DefaultModel = tt.defaultModel
			cfg.providerModelsStore = normalizeProviderModelStore(tt.state, tt.raw)
			cfg.refreshEffectiveProviderModels()

			if got := cfg.GetSelectedModelForProvider(tt.provider); got != tt.want {
				t.Fatalf("GetSelectedModelForProvider(%q) = %q, want %q", tt.provider, got, tt.want)
			}
		})
	}
}
