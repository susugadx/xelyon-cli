package agent

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestGetProviderCompressThreshold_NoDefaultProviderFallback(t *testing.T) {
	cases := []struct {
		name     string
		provider string
		model    string
		want     int
	}{
		{name: "gemini", provider: "gemini", want: 0},
		{name: "anthropic alias", provider: "anthropic", want: 0},
		{name: "deepseek", provider: "deepseek", want: 0},
		{name: "openai legacy", provider: "openai", model: "gpt-5", want: 0},
		{name: "openai gpt-5.4", provider: "openai", model: "gpt-5.4", want: 0},
		{name: "openai gpt-5.4-pro", provider: "openai", model: "gpt-5.4-pro", want: 0},
		{name: "openai gpt-5.4-preview", provider: "openai", model: "gpt-5.4-preview", want: 0},
		{name: "unknown", provider: "unknown", want: 0},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := GetProviderCompressThreshold(tc.provider, tc.model)
			if got != tc.want {
				t.Fatalf("GetProviderCompressThreshold(%q, %q) = %d, want %d", tc.provider, tc.model, got, tc.want)
			}
		})
	}
}

func TestGetProviderCompressThresholdWithConfig_NoDefaultProviderFallback(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		model    string
		want     int
	}{
		{name: "gemini", provider: "gemini", model: "", want: 0},
		{name: "claude", provider: "claude", model: "", want: 0},
		{name: "bedrock", provider: "bedrock", model: "", want: 0},
		{name: "deepseek", provider: "deepseek", model: "", want: 0},
		{name: "openai", provider: "openai", model: "", want: 0},
		{name: "openrouter", provider: "openrouter", model: "", want: 0},
		{name: "unknown", provider: "unknown", model: "", want: 0},
		{name: "empty string", provider: "", model: "", want: 0},
	}

	cfg := config.DefaultConfig()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetProviderCompressThresholdWithConfig(cfg, tt.provider, tt.model)
			if got != tt.want {
				t.Errorf("GetProviderCompressThresholdWithConfig(%q, %q) = %d, want %d", tt.provider, tt.model, got, tt.want)
			}
		})
	}
}

func TestGetProviderCompressThresholdWithConfig_ConfigOverride(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Compression.ProviderThresholds["gemini"] = 200000

	got := GetProviderCompressThresholdWithConfig(cfg, "gemini", "")
	if got != 200000 {
		t.Errorf("config override got %d, want 200000", got)
	}
}

func TestGetProviderCompressThresholdWithConfig_MissingOverrideReturnsZero(t *testing.T) {
	cfg := config.DefaultConfig()

	got := GetProviderCompressThresholdWithConfig(cfg, "gemini", "")
	if got != 0 {
		t.Errorf("missing override got %d, want 0", got)
	}
}

func TestGetProviderCompressThresholdWithConfig_NilConfig(t *testing.T) {
	got := GetProviderCompressThresholdWithConfig(nil, "deepseek", "")
	if got != 0 {
		t.Errorf("nil config got %d, want 0 without explicit provider_thresholds", got)
	}
}

func TestGetProviderCompressThresholdWithConfig_Override(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Compression.ProviderThresholds["deepseek"] = 100000
	cfg.Compression.ProviderThresholds["openai:gpt-5.4"] = 250000
	cfg.Compression.ProviderThresholds["openai:gpt-5.4-preview"] = 240000

	if got := GetProviderCompressThresholdWithConfig(cfg, "deepseek", "deepseek-chat"); got != 100000 {
		t.Fatalf("deepseek override = %d, want 100000", got)
	}
	if got := GetProviderCompressThresholdWithConfig(cfg, "openai", "gpt-5.4"); got != 250000 {
		t.Fatalf("openai model override = %d, want 250000", got)
	}
	if got := GetProviderCompressThresholdWithConfig(cfg, "openai", "gpt-5.4-preview"); got != 240000 {
		t.Fatalf("openai exact preview override = %d, want 240000", got)
	}
	if got := GetProviderCompressThresholdWithConfig(cfg, "openai", "gpt-5.4-preview-lite"); got != 240000 {
		t.Fatalf("openai preview family override = %d, want 240000", got)
	}
}

func TestGetProviderCompressThresholdWithConfig_ModelOverrideBeforeProvider(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Compression.ProviderThresholds["openai"] = 100000
	cfg.Compression.ProviderThresholds["openai:gpt-5.4*"] = 260000

	got := GetProviderCompressThresholdWithConfig(cfg, "openai", "gpt-5.4-preview")
	if got != 260000 {
		t.Errorf("model override got %d, want 260000", got)
	}

	got2 := GetProviderCompressThresholdWithConfig(cfg, "openai", "gpt-5.2")
	if got2 != 100000 {
		t.Errorf("provider fallback got %d, want 100000", got2)
	}
}

func TestGetProviderCompressThresholdWithConfig_ModelFamilyBeforeProvider(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Compression.ProviderThresholds["openai"] = 100000
	cfg.Compression.ProviderThresholds["openai:gpt-5.4"] = 260000

	if got := GetProviderCompressThresholdWithConfig(cfg, "openai", "gpt-5.4-preview"); got != 260000 {
		t.Fatalf("openai family override = %d, want 260000", got)
	}
}

func TestGetProviderCompressThresholdWithConfig_UsesCatalogModelForModelThreshold(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("openai", config.ProviderModelConfig{
		DefaultModel: "corp-gpt-deployment",
		CatalogModel: "gpt-5.4",
	})
	cfg.Compression.ProviderThresholds["openai:gpt-5.4*"] = 260000

	got := GetProviderCompressThresholdWithConfig(cfg, "openai", "corp-gpt-deployment")
	if got != 260000 {
		t.Fatalf("GetProviderCompressThresholdWithConfig(deployment) = %d, want catalog model threshold", got)
	}
}

func TestGetProviderCompressThresholdWithConfig_PrefersExactAnthropicAliasOverride(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Compression.ProviderThresholds["anthropic"] = 123456
	cfg.Compression.ProviderThresholds["claude"] = 654321

	got := GetProviderCompressThresholdWithConfig(cfg, "anthropic", "claude-sonnet-4-6")
	if got != 123456 {
		t.Fatalf("GetProviderCompressThresholdWithConfig() = %d, want %d", got, 123456)
	}
}

func TestGetProviderCompressThresholdWithConfig_PrefersAnthropicAliasModelOverrideBeforeCanonicalFallback(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Compression.ProviderThresholds["anthropic:claude-sonnet-4*"] = 111111
	cfg.Compression.ProviderThresholds["claude:claude-sonnet-4*"] = 222222

	got := GetProviderCompressThresholdWithConfig(cfg, "anthropic", "claude-sonnet-4-6")
	if got != 111111 {
		t.Fatalf("GetProviderCompressThresholdWithConfig() = %d, want %d", got, 111111)
	}
}

func TestGetProviderCompressThresholdWithConfig_FallsBackToCanonicalWhenExactAnthropicAliasMissing(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Compression.ProviderThresholds["claude"] = 222222

	got := GetProviderCompressThresholdWithConfig(cfg, "anthropic", "claude-sonnet-4-6")
	if got != 222222 {
		t.Fatalf("GetProviderCompressThresholdWithConfig() = %d, want %d", got, 222222)
	}
}

func TestGetProviderCompressThresholdWithConfig_CanonicalClaudeFallsBackToAnthropicAliasOverride(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Compression.ProviderThresholds = map[string]int{
		"anthropic": 123456,
	}

	got := GetProviderCompressThresholdWithConfig(cfg, "claude", "claude-sonnet-4-6")
	if got != 123456 {
		t.Fatalf("GetProviderCompressThresholdWithConfig() = %d, want %d", got, 123456)
	}
}
