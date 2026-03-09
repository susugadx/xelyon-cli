package api

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestGetDefaultModel_UsesGlobalFallbackContext(t *testing.T) {
	original := config.GetGlobalConfig()
	cfg := config.DefaultConfig()
	cfg.ProviderModels["openai"] = config.ProviderModelConfig{DefaultModel: "gpt-5.2-runtime"}
	config.SetGlobalConfig(cfg)
	t.Cleanup(func() {
		config.SetGlobalConfig(original)
	})

	got := GetDefaultModel("", "openai", "fallback")
	if got != "gpt-5.2-runtime" {
		t.Fatalf("GetDefaultModel() = %q, want %q", got, "gpt-5.2-runtime")
	}
}
