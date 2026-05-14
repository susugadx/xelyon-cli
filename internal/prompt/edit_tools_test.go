package prompt

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestResolveEditToolMode_ProviderAllowlistAndLegacyFallback(t *testing.T) {
	t.Setenv("XELYON_EDIT_TOOL", "")

	tests := []struct {
		name     string
		provider string
		model    string
		want     EditToolMode
	}{
		{name: "openai", provider: "openai", model: "gpt-5.4", want: EditToolModeApplyPatch},
		{name: "azure", provider: "azure", model: "azure-gpt-5.4", want: EditToolModeApplyPatch},
		{name: "gemini", provider: "gemini", model: "gemini-3.1-pro", want: EditToolModeApplyPatch},
		{name: "google", provider: "google", model: "gemini-3.1-pro", want: EditToolModeApplyPatch},
		{name: "kimi", provider: "kimi", model: "kimi-k2.6", want: EditToolModeLegacy},
		{name: "moonshot alias", provider: "moonshot", model: "kimi-k2.6", want: EditToolModeLegacy},
		{name: "claude", provider: "claude", model: "claude-sonnet-4-6", want: EditToolModeLegacy},
		{name: "anthropic alias", provider: "anthropic", model: "claude-sonnet-4-6", want: EditToolModeLegacy},
		{name: "deepseek", provider: "deepseek", model: "deepseek-chat", want: EditToolModeLegacy},
		{name: "groq", provider: "groq", model: "meta-llama/llama-4-scout-17b-16e-instruct", want: EditToolModeLegacy},
		{name: "ollama", provider: "ollama", model: "qwen2.5-coder:7b", want: EditToolModeLegacy},
		{name: "unknown provider", provider: "custom", model: "custom-model", want: EditToolModeLegacy},
		{name: "openrouter openai", provider: "openrouter", model: "openai/gpt-5.4", want: EditToolModeApplyPatch},
		{name: "openrouter google", provider: "openrouter", model: "google/gemini-2.5-pro", want: EditToolModeApplyPatch},
		{name: "openrouter gemini", provider: "openrouter", model: "gemini/gemini-3.1-pro", want: EditToolModeApplyPatch},
		{name: "openrouter anthropic", provider: "openrouter", model: "anthropic/claude-sonnet-4.6", want: EditToolModeLegacy},
		{name: "openrouter deepseek", provider: "openrouter", model: "deepseek/deepseek-v4-flash", want: EditToolModeLegacy},
		{name: "openrouter moonshotai", provider: "openrouter", model: "moonshotai/kimi-k2.6", want: EditToolModeLegacy},
		{name: "openrouter unknown family", provider: "openrouter", model: "meta-llama/llama-4-scout-17b-16e-instruct", want: EditToolModeLegacy},
		{name: "bedrock claude family", provider: "bedrock", model: "global.anthropic.claude-sonnet-4-6", want: EditToolModeLegacy},
		{name: "bedrock non-claude", provider: "bedrock", model: "amazon.nova-pro-v1:0", want: EditToolModeLegacy},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ResolveEditToolMode(tt.provider, tt.model); got != tt.want {
				t.Fatalf("ResolveEditToolMode(%q, %q) = %q, want %q", tt.provider, tt.model, got, tt.want)
			}
		})
	}
}

func TestResolveEditToolMode_EnvOverride(t *testing.T) {
	t.Setenv("XELYON_EDIT_TOOL", "legacy")

	if got := ResolveEditToolMode("openai", "gpt-5.4"); got != EditToolModeLegacy {
		t.Fatalf("ResolveEditToolMode() with env override = %q, want %q", got, EditToolModeLegacy)
	}
}

func TestResolveEditToolModeWithConfig_UsesRuntimeModelInsteadOfCatalogAlias(t *testing.T) {
	t.Setenv("XELYON_EDIT_TOOL", "")

	cfg := config.DefaultConfig()
	if cfg.ProviderModels == nil {
		cfg.ProviderModels = map[string]config.ProviderModelConfig{}
	}
	cfg.ProviderModels["openrouter"] = config.ProviderModelConfig{
		DefaultModel: "moonshotai/kimi-k2.6",
		CatalogModel: "openai/gpt-5.4",
	}

	if got := ResolveEditToolModeWithConfig("openrouter", "moonshotai/kimi-k2.6", cfg); got != EditToolModeLegacy {
		t.Fatalf("ResolveEditToolModeWithConfig(openrouter catalog alias) = %q, want %q", got, EditToolModeLegacy)
	}
}
