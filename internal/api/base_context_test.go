package api

import (
	"context"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestGetDefaultModelWithContext_UsesInjectedConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.ProviderModels["openai"] = config.ProviderModelConfig{DefaultModel: "gpt-5.2-runtime"}
	ctx := config.WithContext(context.Background(), cfg)

	got := GetDefaultModelWithContext(ctx, "", "openai", "fallback")
	if got != "gpt-5.2-runtime" {
		t.Fatalf("GetDefaultModelWithContext() = %q, want %q", got, "gpt-5.2-runtime")
	}
}
