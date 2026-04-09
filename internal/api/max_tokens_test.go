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
			expected: 8192,
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
