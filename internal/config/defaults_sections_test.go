package config

import "testing"

func TestDefaultConfig_CollectionFieldsAreIndependentPerCall(t *testing.T) {
	cfg1 := DefaultConfig()
	cfg2 := DefaultConfig()

	cfg1.CommandAliases["alias-a"] = "config"
	cfg1.ProviderModels["openai"] = ProviderModelConfig{DefaultModel: "custom-openai"}
	cfg1.Compression.ProviderThresholds["custom"] = 123
	cfg1.LSP.Servers["custom"] = LSPServerConfig{Command: "custom-lsp"}

	if _, ok := cfg2.CommandAliases["alias-a"]; ok {
		t.Fatal("CommandAliases should not be shared across DefaultConfig() calls")
	}
	if got := cfg2.ProviderModels["openai"].DefaultModel; got != "gpt-5.4" {
		t.Fatalf("ProviderModels[openai].DefaultModel = %q, want default %q", got, "gpt-5.4")
	}
	if got := len(cfg2.Compression.ProviderThresholds); got != 0 {
		t.Fatalf("Compression.ProviderThresholds default len = %d, want 0", got)
	}
	if _, ok := cfg2.Compression.ProviderThresholds["custom"]; ok {
		t.Fatal("Compression.ProviderThresholds should not be shared across DefaultConfig() calls")
	}
	if _, ok := cfg2.LSP.Servers["custom"]; ok {
		t.Fatal("LSP.Servers should not be shared across DefaultConfig() calls")
	}
}

func TestDefaultConfig_ProviderModelStoreStateIsInMemoryEffectiveOnly(t *testing.T) {
	cfg := DefaultConfig()

	if got := cfg.providerModelSectionState(); got != providerModelSectionStateInMemoryEffectiveOnly {
		t.Fatalf("providerModelSectionState() = %v, want %v", got, providerModelSectionStateInMemoryEffectiveOnly)
	}
}
