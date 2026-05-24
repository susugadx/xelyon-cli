package providerdiag

import (
	"context"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestResolveProviderDiagnosticModelUsesSharedPrecedence(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("deepseek", config.ProviderModelConfig{
		DefaultModel: "configured-deepseek",
	})

	t.Setenv("XELYON_MODEL", "env-model")
	if got, source := ResolveProviderDiagnosticModel(cfg, "deepseek", "explicit-model", "fallback-model"); got != "explicit-model" || source != "--model" {
		t.Fatalf("explicit model = %q (%s), want explicit-model (--model)", got, source)
	}

	if got, source := ResolveProviderDiagnosticModel(cfg, "deepseek", "", "fallback-model"); got != "env-model" || source != "XELYON_MODEL" {
		t.Fatalf("env model = %q (%s), want env-model (XELYON_MODEL)", got, source)
	}

	t.Setenv("XELYON_MODEL", "")
	if got, source := ResolveProviderDiagnosticModel(cfg, "deepseek", "", "fallback-model"); got != "configured-deepseek" || source != "provider_models.deepseek.default_model" {
		t.Fatalf("configured model = %q (%s), want configured-deepseek (provider default)", got, source)
	}
}

func TestResolveProviderDiagnosticCatalogModelUsesSharedPrecedence(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("deepseek", config.ProviderModelConfig{
		DefaultModel: "corp-deepseek",
		CatalogModel: "deepseek-v4-flash",
		ModelOverrides: map[string]config.ModelOverride{
			"other-deepseek": {
				CatalogModel: "deepseek-v4-pro",
			},
		},
	})

	if got, source := ResolveProviderDiagnosticCatalogModel(cfg, "deepseek", "corp-deepseek", "explicit-catalog"); got != "explicit-catalog" || source != "--catalog-model" {
		t.Fatalf("explicit catalog = %q (%s), want explicit-catalog (--catalog-model)", got, source)
	}
	if got, source := ResolveProviderDiagnosticCatalogModel(cfg, "deepseek", "other-deepseek", ""); got != "deepseek-v4-pro" || source != "provider_models.deepseek.model_overrides" {
		t.Fatalf("override catalog = %q (%s), want deepseek-v4-pro (model_overrides)", got, source)
	}
	if got, source := ResolveProviderDiagnosticCatalogModel(cfg, "deepseek", "corp-deepseek", ""); got != "deepseek-v4-flash" || source != "provider_models.deepseek.catalog_model" {
		t.Fatalf("provider catalog = %q (%s), want deepseek-v4-flash (provider catalog_model)", got, source)
	}
	if got, source := ResolveProviderDiagnosticCatalogModel(cfg, "deepseek", "unconfigured-model", ""); got != "unconfigured-model" || source != "model" {
		t.Fatalf("plain catalog = %q (%s), want unconfigured-model (model)", got, source)
	}
}

func TestResolveProviderDiagnosticCatalogModelCanonicalizesGeminiResourceNames(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("gemini", config.ProviderModelConfig{
		DefaultModel: "corp-gemini",
		CatalogModel: "models/gemini-3.5-flash",
	})

	if got, source := ResolveProviderDiagnosticCatalogModel(cfg, "gemini", "corp-gemini", ""); got != "gemini-3.5-flash" || source != "provider_models.gemini.catalog_model" {
		t.Fatalf("configured Gemini catalog = %q (%s), want canonical catalog model", got, source)
	}
	if got, source := ResolveProviderDiagnosticCatalogModel(cfg, "gemini", "corp-gemini", "models/gemini-2.0-flash-lite"); got != "gemini-2.0-flash-lite" || source != "--catalog-model" {
		t.Fatalf("explicit Gemini catalog = %q (%s), want canonical explicit catalog model", got, source)
	}
	if !IsProviderCatalogModelKnown("gemini", "models/gemini-3.5-flash") {
		t.Fatal("IsProviderCatalogModelKnown(gemini, models/gemini-3.5-flash) = false, want true")
	}
}

func TestProviderDiagnosticPolicyConfigKeepsProviderScopedCatalogPolicy(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("deepseek", config.ProviderModelConfig{
		DefaultModel: "corp-deepseek",
		CatalogModel: "deepseek-v4-flash",
	})

	invalidCatalogCfg := ProviderDiagnosticPolicyConfig(cfg, ProviderDiagnosticPolicyConfigOptions{
		Provider:     "deepseek",
		Model:        "corp-deepseek",
		CatalogModel: "gpt-5.5",
	})
	if got := invalidCatalogCfg.ModelCatalogName("deepseek", "corp-deepseek"); got != "corp-deepseek" {
		t.Fatalf("invalid catalog policy model = %q, want request model with non-DeepSeek catalog cleared", got)
	}

	validCatalogCfg := ProviderDiagnosticPolicyConfig(cfg, ProviderDiagnosticPolicyConfigOptions{
		Provider:        "deepseek",
		Model:           "corp-deepseek",
		CatalogModel:    "deepseek-v4-pro",
		MaxOutputTokens: 64,
	})
	if got := validCatalogCfg.ModelCatalogName("deepseek", "corp-deepseek"); got != "deepseek-v4-pro" {
		t.Fatalf("valid catalog policy model = %q, want deepseek-v4-pro", got)
	}
	if got := api.GetMaxOutputTokens(config.WithContext(context.Background(), validCatalogCfg), "deepseek", "corp-deepseek"); got != 64 {
		t.Fatalf("GetMaxOutputTokens(valid catalog policy) = %d, want explicit smoke max output", got)
	}
}

func TestProviderDiagnosticPolicyConfigCanonicalizesGeminiResourceCatalogModel(t *testing.T) {
	cfg := ProviderDiagnosticPolicyConfig(config.DefaultConfig(), ProviderDiagnosticPolicyConfigOptions{
		Provider:     "gemini",
		Model:        "corp-gemini",
		CatalogModel: "models/gemini-3.5-flash",
	})
	if got := cfg.ModelCatalogName("gemini", "corp-gemini"); got != "gemini-3.5-flash" {
		t.Fatalf("Gemini diagnostic policy catalog = %q, want gemini-3.5-flash", got)
	}
}
