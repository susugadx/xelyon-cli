package config

import "testing"

func TestProviderModelWriteKey_PrefersAliasOverrideOverCanonicalDefault(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DefaultProvider = "deepseek"
	cfg.SetProviderModelConfig("anthropic", ProviderModelConfig{DefaultModel: "anthropic-custom"})

	key, ok := cfg.ProviderModelWriteKey("claude")
	if !ok {
		t.Fatal("ProviderModelWriteKey(claude) should succeed")
	}
	if key != "anthropic" {
		t.Fatalf("ProviderModelWriteKey(claude) = %q, want %q", key, "anthropic")
	}
}

func TestProviderModelWriteKey_PrefersDefaultProviderAliasWhenPresent(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DefaultProvider = "anthropic"
	cfg.SetProviderModelConfig("anthropic", ProviderModelConfig{DefaultModel: "anthropic-custom"})

	key, ok := cfg.ProviderModelWriteKey("claude")
	if !ok {
		t.Fatal("ProviderModelWriteKey(claude) should succeed")
	}
	if key != "anthropic" {
		t.Fatalf("ProviderModelWriteKey(claude) = %q, want %q", key, "anthropic")
	}
}

func TestProviderModelWriteKey_CreatesExactRequestedAliasWhenNoRawEntryExists(t *testing.T) {
	cfg := DefaultConfig()
	cfg.DefaultProvider = "claude"

	key, ok := cfg.ProviderModelWriteKey("anthropic")
	if !ok {
		t.Fatal("ProviderModelWriteKey(anthropic) should succeed")
	}
	if key != "anthropic" {
		t.Fatalf("ProviderModelWriteKey(anthropic) = %q, want %q", key, "anthropic")
	}
}

func TestProviderModelWriteKey_ExplicitExactKeyWinsAtDefaultValue(t *testing.T) {
	cfg := DefaultConfig()
	cfg.SetProviderModelsForEdit(map[string]ProviderModelConfig{
		"anthropic": {DefaultModel: "anthropic-custom"},
		"claude": {
			DefaultModel: DefaultConfig().ProviderModels["claude"].DefaultModel,
		},
	})

	key, ok := cfg.ProviderModelWriteKey("claude")
	if !ok {
		t.Fatal("ProviderModelWriteKey(claude) should succeed")
	}
	if key != "claude" {
		t.Fatalf("ProviderModelWriteKey(claude) = %q, want %q", key, "claude")
	}

	if key, ok = cfg.ProviderModelWriteKey("anthropic"); !ok {
		t.Fatal("ProviderModelWriteKey(anthropic) should succeed")
	} else if key != "anthropic" {
		t.Fatalf("ProviderModelWriteKey(anthropic) = %q, want %q", key, "anthropic")
	}
}
