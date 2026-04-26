package api

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestGetMaxOutputTokens(t *testing.T) {
	cfg := &config.Config{
		ProviderModels: map[string]config.ProviderModelConfig{
			"deepseek": {
				MaxOutputTokens: 16384,
				ModelOverrides: map[string]config.ModelOverride{
					"user-model": {MaxOutputTokens: 9999},
				},
			},
		},
	}

	tests := []struct {
		name     string
		provider string
		model    string
		expected int
	}{
		{
			name:     "User override has highest priority",
			provider: "deepseek",
			model:    "user-model",
			expected: 9999,
		},
		{
			name:     "Known model map has second priority",
			provider: "deepseek",
			model:    "deepseek-chat",
			expected: 384000,
		},
		{
			name:     "Provider default is fallback",
			provider: "deepseek",
			model:    "unknown-model",
			expected: 16384,
		},
		{
			name:     "Works even if provider config is missing",
			provider: "unknown-provider",
			model:    "claude-sonnet-4-6",
			expected: 64000,
		},
		{
			name:     "Returns 0 for completely unknown",
			provider: "unknown-provider",
			model:    "totally-unknown",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := config.WithContext(context.Background(), cfg)
			got := GetMaxOutputTokens(ctx, tt.provider, tt.model)
			if got != tt.expected {
				t.Errorf("GetMaxOutputTokens() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetMaxOutputTokens_UsesCatalogModelForDeploymentName(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("deepseek", config.ProviderModelConfig{
		DefaultModel: "corp-chat-deployment",
		CatalogModel: "deepseek-chat",
		ModelOverrides: map[string]config.ModelOverride{
			"reasoner-deployment": {CatalogModel: "deepseek-reasoner"},
			"manual-limit": {
				CatalogModel:    "deepseek-chat",
				MaxOutputTokens: 7777,
			},
		},
	})
	ctx := config.WithContext(context.Background(), cfg)

	if got := GetMaxOutputTokens(ctx, "deepseek", "corp-chat-deployment"); got != 384000 {
		t.Fatalf("GetMaxOutputTokens(default deployment) = %d, want 384000 from deepseek-chat", got)
	}
	if got := GetMaxOutputTokens(ctx, "deepseek", "reasoner-deployment"); got != 384000 {
		t.Fatalf("GetMaxOutputTokens(reasoner deployment) = %d, want 384000 from deepseek-reasoner", got)
	}
	if got := GetMaxOutputTokens(ctx, "deepseek", "manual-limit"); got != 7777 {
		t.Fatalf("GetMaxOutputTokens(manual limit) = %d, want explicit override", got)
	}
}

func TestGetMaxOutputTokens_DeepSeekV4LimitDoesNotLeakToPassThroughModels(t *testing.T) {
	cfg := config.DefaultConfig()
	ctx := config.WithContext(context.Background(), cfg)

	if got := GetMaxOutputTokens(ctx, "deepseek", "deepseek-v4-custom"); got != 384000 {
		t.Fatalf("GetMaxOutputTokens(deepseek-v4-custom) = %d, want 384000 from V4 family", got)
	}
	if got := GetMaxOutputTokens(ctx, "deepseek", "deepseek-coder"); got != 16384 {
		t.Fatalf("GetMaxOutputTokens(deepseek-coder) = %d, want conservative pass-through limit", got)
	}
	if got := GetMaxOutputTokens(ctx, "deepseek", "unknown-model"); got != 16384 {
		t.Fatalf("GetMaxOutputTokens(unknown-model) = %d, want provider fallback", got)
	}
}

func TestGetMaxOutputTokens_AnthropicAliasConfigAppliesToClaudeRuntime(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("anthropic", config.ProviderModelConfig{MaxOutputTokens: 12345})

	ctx := config.WithContext(context.Background(), cfg)
	got := GetMaxOutputTokens(ctx, "claude", "alias-unknown-model")
	if got != 12345 {
		t.Fatalf("GetMaxOutputTokens() = %v, want %v", got, 12345)
	}
}

func TestGetMaxOutputTokens_PrefersRequestedAnthropicAliasWhenBothAliasEntriesExist(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".xelyon")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	yamlData := `
default_provider: deepseek
provider_models:
  anthropic:
    max_output_tokens: 12345
  claude:
    max_output_tokens: 54321
`
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(yamlData), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	ctx := config.WithContext(context.Background(), cfg)
	got := GetMaxOutputTokens(ctx, "anthropic", "alias-unknown-model")
	if got != 12345 {
		t.Fatalf("GetMaxOutputTokens() = %v, want %v", got, 12345)
	}
}

func TestGetMaxOutputTokens_ClaudeRuntimeUsesAnthropicOwnerForAnthropicSelectedModel(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".xelyon")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	yamlData := `
default_provider: deepseek
provider_models:
  anthropic:
    default_model: anthropic-custom
    max_output_tokens: 12345
  claude:
    default_model: claude-custom
    max_output_tokens: 54321
`
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(yamlData), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	ctx := config.WithContext(context.Background(), cfg)
	got := GetMaxOutputTokens(ctx, "claude", "anthropic-custom")
	if got != 12345 {
		t.Fatalf("GetMaxOutputTokens() = %v, want %v", got, 12345)
	}
}

func TestGetMaxOutputTokens_PrefersRequestedAnthropicAliasWhenBothAliasEntriesShareSameModel(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".xelyon")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	yamlData := `
default_provider: deepseek
provider_models:
  anthropic:
    default_model: shared-custom
    max_output_tokens: 12345
  claude:
    default_model: shared-custom
    max_output_tokens: 54321
`
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(yamlData), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	ctx := config.WithContext(context.Background(), cfg)
	got := GetMaxOutputTokens(ctx, "anthropic", "shared-custom")
	if got != 12345 {
		t.Fatalf("GetMaxOutputTokens() = %v, want %v", got, 12345)
	}
}

func TestGetMaxOutputTokens_PrefersActiveAnthropicAliasForCustomClaudeModelWithoutSelectedOwner(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("HOME", tmpDir)

	configDir := filepath.Join(tmpDir, ".xelyon")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	yamlData := `
default_provider: claude
provider_models:
  anthropic:
    max_output_tokens: 12345
    model_overrides:
      corp-custom:
        max_output_tokens: 11111
  claude:
    max_output_tokens: 54321
    model_overrides:
      corp-custom:
        max_output_tokens: 22222
`
	configPath := filepath.Join(configDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte(yamlData), 0600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	ctx := config.WithContext(context.Background(), cfg)
	got := GetMaxOutputTokens(ctx, "anthropic", "corp-custom")
	if got != 11111 {
		t.Fatalf("GetMaxOutputTokens() = %v, want %v", got, 11111)
	}
}
