package agent

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

// --- defaultProviderCompressThreshold tests ---

func TestDefaultProviderCompressThreshold_AllProviders(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		model    string
		want     int
	}{
		{name: "gemini", provider: "gemini", model: "", want: 180000},
		{name: "claude", provider: "claude", model: "", want: 150000},
		{name: "bedrock", provider: "bedrock", model: "", want: 150000},
		{name: "deepseek", provider: "deepseek", model: "", want: 600000},
		{name: "openai", provider: "openai", model: "", want: 0},
		{name: "openrouter", provider: "openrouter", model: "", want: 120000},
		{name: "unknown", provider: "unknown", model: "", want: 0},
		{name: "empty string", provider: "", model: "", want: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := defaultProviderCompressThreshold(tt.provider, tt.model)
			if got != tt.want {
				t.Errorf("defaultProviderCompressThreshold(%q, %q) = %d, want %d", tt.provider, tt.model, got, tt.want)
			}
		})
	}
}

// --- lookupConfiguredModelThreshold tests ---

func TestLookupConfiguredModelThreshold_ExactMatch(t *testing.T) {
	thresholds := map[string]int{
		"openai:gpt-5.4": 260000,
	}
	got, ok := lookupConfiguredModelThreshold(thresholds, "openai", "gpt-5.4")
	if !ok {
		t.Fatal("expected exact match to succeed")
	}
	if got != 260000 {
		t.Errorf("got %d, want 260000", got)
	}
}

func TestLookupConfiguredModelThreshold_WildcardMatch(t *testing.T) {
	thresholds := map[string]int{
		"openai:gpt-5.4*": 260000,
	}
	got, ok := lookupConfiguredModelThreshold(thresholds, "openai", "gpt-5.4-preview")
	if !ok {
		t.Fatal("expected wildcard match to succeed")
	}
	if got != 260000 {
		t.Errorf("got %d, want 260000", got)
	}
}

func TestLookupConfiguredModelThreshold_LongestPrefixWins(t *testing.T) {
	thresholds := map[string]int{
		"openai:gpt-5*":       100000,
		"openai:gpt-5.4*":     260000,
		"openai:gpt-5.4-pro*": 270000,
	}

	// "gpt-5.4-pro-latest" should match "openai:gpt-5.4-pro*" (longest prefix)
	got, ok := lookupConfiguredModelThreshold(thresholds, "openai", "gpt-5.4-pro-latest")
	if !ok {
		t.Fatal("expected longest prefix match to succeed")
	}
	if got != 270000 {
		t.Errorf("got %d, want 270000 (longest prefix)", got)
	}

	// "gpt-5.4-mini" should match "openai:gpt-5.4*"
	got2, ok2 := lookupConfiguredModelThreshold(thresholds, "openai", "gpt-5.4-mini")
	if !ok2 {
		t.Fatal("expected wildcard match for gpt-5.4-mini to succeed")
	}
	if got2 != 260000 {
		t.Errorf("got %d, want 260000", got2)
	}
}

func TestLookupConfiguredModelThreshold_NoMatch(t *testing.T) {
	thresholds := map[string]int{
		"openai:gpt-5.4*": 260000,
	}
	_, ok := lookupConfiguredModelThreshold(thresholds, "openai", "gpt-5.2")
	if ok {
		t.Error("expected no match for gpt-5.2 against gpt-5.4*")
	}
}

func TestLookupConfiguredModelThreshold_EmptyThresholds(t *testing.T) {
	_, ok := lookupConfiguredModelThreshold(nil, "openai", "gpt-5.4")
	if ok {
		t.Error("expected no match with nil thresholds")
	}

	_, ok = lookupConfiguredModelThreshold(map[string]int{}, "openai", "gpt-5.4")
	if ok {
		t.Error("expected no match with empty thresholds")
	}
}

func TestLookupConfiguredModelThreshold_EmptyProviderOrModel(t *testing.T) {
	thresholds := map[string]int{
		"openai:gpt-5.4": 260000,
	}
	_, ok := lookupConfiguredModelThreshold(thresholds, "", "gpt-5.4")
	if ok {
		t.Error("expected no match with empty provider")
	}

	_, ok = lookupConfiguredModelThreshold(thresholds, "openai", "")
	if ok {
		t.Error("expected no match with empty model")
	}
}

// --- GetProviderCompressThresholdWithConfig tests ---

func TestGetProviderCompressThresholdWithConfig_ConfigOverride(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Compression.ProviderThresholds["gemini"] = 200000

	got := GetProviderCompressThresholdWithConfig(cfg, "gemini", "")
	if got != 200000 {
		t.Errorf("config override got %d, want 200000", got)
	}
}

func TestGetProviderCompressThresholdWithConfig_DefaultFallback(t *testing.T) {
	cfg := config.DefaultConfig()
	// Remove the configured threshold to force default
	delete(cfg.Compression.ProviderThresholds, "gemini")

	got := GetProviderCompressThresholdWithConfig(cfg, "gemini", "")
	if got != 180000 {
		t.Errorf("default fallback got %d, want 180000", got)
	}
}

func TestGetProviderCompressThresholdWithConfig_NilConfig(t *testing.T) {
	// nil config should use DefaultConfig
	got := GetProviderCompressThresholdWithConfig(nil, "deepseek", "")
	if got == 0 {
		t.Error("nil config should use defaults, not return 0 for deepseek")
	}
}

func TestGetProviderCompressThresholdWithConfig_ModelOverrideBeforeProvider(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Compression.ProviderThresholds["openai"] = 100000
	cfg.Compression.ProviderThresholds["openai:gpt-5.4*"] = 260000

	// Model-specific wildcard should win over provider-level
	got := GetProviderCompressThresholdWithConfig(cfg, "openai", "gpt-5.4-preview")
	if got != 260000 {
		t.Errorf("model override got %d, want 260000", got)
	}

	// Model without wildcard match falls to provider-level
	got2 := GetProviderCompressThresholdWithConfig(cfg, "openai", "gpt-5.2")
	if got2 != 100000 {
		t.Errorf("provider fallback got %d, want 100000", got2)
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

// --- averageOutputTokens tests ---

func TestAverageOutputTokens_Normal(t *testing.T) {
	stats := &SessionStats{
		OutputTokens:      3000,
		AssistantMessages: 3,
	}
	got := averageOutputTokens(stats)
	if got != 1000 {
		t.Errorf("averageOutputTokens() = %d, want 1000", got)
	}
}

func TestAverageOutputTokens_NilStats(t *testing.T) {
	got := averageOutputTokens(nil)
	if got != 0 {
		t.Errorf("averageOutputTokens(nil) = %d, want 0", got)
	}
}

func TestAverageOutputTokens_ZeroOutput(t *testing.T) {
	stats := &SessionStats{
		OutputTokens:      0,
		AssistantMessages: 5,
	}
	got := averageOutputTokens(stats)
	if got != 0 {
		t.Errorf("averageOutputTokens() with zero output = %d, want 0", got)
	}
}

func TestAverageOutputTokens_ZeroMessages(t *testing.T) {
	stats := &SessionStats{
		OutputTokens:      3000,
		AssistantMessages: 0,
	}
	// Should not divide by zero; AssistantMessages clamped to 1
	got := averageOutputTokens(stats)
	if got != 3000 {
		t.Errorf("averageOutputTokens() with zero messages = %d, want 3000", got)
	}
}

// --- shouldForceCompressForPricingCliff tests ---

func TestShouldForceCompressForPricingCliff_ZeroTokens(t *testing.T) {
	projected, force := shouldForceCompressForPricingCliff("claude", "claude-sonnet-4-5", 0, nil)
	if force {
		t.Error("should not force compress with zero tokens")
	}
	if projected != 0 {
		t.Errorf("projected = %d, want 0", projected)
	}
}

func TestShouldForceCompressForPricingCliff_NegativeTokens(t *testing.T) {
	projected, force := shouldForceCompressForPricingCliff("claude", "claude-sonnet-4-5", -100, nil)
	if force {
		t.Error("should not force compress with negative tokens")
	}
	if projected != -100 {
		t.Errorf("projected = %d, want -100", projected)
	}
}

func TestShouldForceCompressForPricingCliff_WellBelowCliff(t *testing.T) {
	stats := &SessionStats{
		OutputTokens:      2000,
		AssistantMessages: 1,
	}
	_, force := shouldForceCompressForPricingCliff("claude", "claude-sonnet-4-5", 100000, stats)
	if force {
		t.Error("should not force compress well below 200K cliff")
	}
}

func TestShouldForceCompressForPricingCliff_AtCliffBoundary(t *testing.T) {
	stats := &SessionStats{
		OutputTokens:      5000,
		AssistantMessages: 1,
	}
	// 199000 + 5000 avg = 204000 > 200K cliff
	projected, force := shouldForceCompressForPricingCliff("claude", "claude-sonnet-4-5", 199000, stats)
	if !force {
		t.Error("should force compress when projected exceeds 200K cliff")
	}
	if projected != 204000 {
		t.Errorf("projected = %d, want 204000", projected)
	}
}

func TestShouldForceCompressForPricingCliffForConfig_UsesCatalogModel(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("openai", config.ProviderModelConfig{
		DefaultModel: "corp-gpt-deployment",
		CatalogModel: "gpt-5.4",
	})
	stats := &SessionStats{OutputTokens: 2000, AssistantMessages: 1}

	projected, force := shouldForceCompressForPricingCliffForConfig(cfg, "openai", "corp-gpt-deployment", 271000, stats)
	if projected != 273000 {
		t.Fatalf("projected = %d, want 273000", projected)
	}
	if !force {
		t.Fatal("force = false, want true from gpt-5.4 pricing cliff")
	}
}
