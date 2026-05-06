package config

import "testing"

func TestGetModelForProvider(t *testing.T) {
	cfg := DefaultConfig()

	tests := []struct {
		name     string
		provider string
		want     string
	}{
		{name: "deepseek", provider: "deepseek", want: "deepseek-v4-flash"},
		{name: "openai", provider: "openai", want: "gpt-5.4"},
		{name: "kimi", provider: "kimi", want: "kimi-k2.6"},
		{name: "moonshot alias", provider: "moonshot", want: "kimi-k2.6"},
		{name: "azure", provider: "azure", want: "azure-gpt-5.4"},
		{name: "claude", provider: "claude", want: "claude-sonnet-4-6"},
		{name: "anthropic alias", provider: "anthropic", want: "claude-sonnet-4-6"},
		{name: "ollama", provider: "ollama", want: "qwen2.5-coder:7b"},
		{name: "groq", provider: "groq", want: "meta-llama/llama-4-scout-17b-16e-instruct"},
		{name: "unknown", provider: "unknown", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cfg.GetModelForProvider(tt.provider)
			if got != tt.want {
				t.Errorf("GetModelForProvider(%q) = %q, want %q", tt.provider, got, tt.want)
			}
		})
	}
}

func TestValidateModelForProvider(t *testing.T) {
	cfg := DefaultConfig()

	tests := []struct {
		name     string
		provider string
		model    string
		want     bool
	}{
		{name: "valid provider", provider: "deepseek", model: "any-model", want: true},
		{name: "kimi provider", provider: "kimi", model: "kimi-k2.6", want: true},
		{name: "moonshot alias", provider: "moonshot", model: "kimi-k2.6", want: true},
		{name: "anthropic alias", provider: "anthropic", model: "any-model", want: true},
		{name: "invalid provider", provider: "unknown", model: "any-model", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cfg.ValidateModelForProvider(tt.provider, tt.model)
			if got != tt.want {
				t.Errorf("ValidateModelForProvider(%q, %q) = %v, want %v", tt.provider, tt.model, got, tt.want)
			}
		})
	}
}
