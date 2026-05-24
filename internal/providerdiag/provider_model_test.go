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

func TestModelGatedImageInputAvailability(t *testing.T) {
	tests := []struct {
		name             string
		provider         string
		model            string
		catalogModel     string
		providerPath     bool
		wantAvailability CapabilityAvailability
	}{
		{
			name:             "trusted openrouter delegated vision catalog",
			provider:         "openrouter",
			model:            "corp-openrouter-model",
			catalogModel:     "openai/gpt-5.4",
			providerPath:     true,
			wantAvailability: KnownCapabilityAvailability(true),
		},
		{
			name:             "trusted openrouter delegated non image catalog",
			provider:         "openrouter",
			model:            "corp-openrouter-model",
			catalogModel:     "deepseek/deepseek-v4-flash",
			providerPath:     true,
			wantAvailability: KnownCapabilityAvailability(false),
		},
		{
			name:             "untrusted openrouter catalog",
			provider:         "openrouter",
			model:            "corp-openrouter-model",
			catalogModel:     "vendor/model",
			providerPath:     true,
			wantAvailability: UnknownCapabilityAvailability(),
		},
		{
			name:             "azure trusts openai catalog model",
			provider:         "azure",
			model:            "corp-deployment",
			catalogModel:     "gpt-5.4",
			providerPath:     true,
			wantAvailability: KnownCapabilityAvailability(true),
		},
		{
			name:             "provider path disabled",
			provider:         "openrouter",
			model:            "openai/gpt-5.4",
			catalogModel:     "openai/gpt-5.4",
			providerPath:     false,
			wantAvailability: KnownCapabilityAvailability(false),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ModelGatedImageInputAvailability(tt.provider, tt.model, tt.catalogModel, tt.providerPath)
			if got != tt.wantAvailability {
				t.Fatalf("ModelGatedImageInputAvailability() = %+v, want %+v", got, tt.wantAvailability)
			}
		})
	}
}

func TestModelGatedWebSearchAvailability(t *testing.T) {
	tests := []struct {
		name             string
		provider         string
		model            string
		catalogModel     string
		providerPath     bool
		wantAvailability CapabilityAvailability
	}{
		{
			name:             "trusted claude catalog",
			provider:         "claude",
			model:            "corp-claude-model",
			catalogModel:     "claude-sonnet-4-6",
			providerPath:     true,
			wantAvailability: KnownCapabilityAvailability(true),
		},
		{
			name:             "untrusted claude catalog",
			provider:         "claude",
			model:            "corp-claude-model",
			catalogModel:     "gpt-5.4",
			providerPath:     true,
			wantAvailability: UnknownCapabilityAvailability(),
		},
		{
			name:             "trusted gemini catalog",
			provider:         "gemini",
			model:            "corp-gemini-model",
			catalogModel:     "gemini-3.1-pro-preview-customtools",
			providerPath:     true,
			wantAvailability: KnownCapabilityAvailability(true),
		},
		{
			name:             "gemini pricing-only model is not trusted",
			provider:         "gemini",
			model:            "gemini-pro",
			catalogModel:     "gemini-pro",
			providerPath:     true,
			wantAvailability: UnknownCapabilityAvailability(),
		},
		{
			name:             "provider path disabled",
			provider:         "claude",
			model:            "claude-sonnet-4-6",
			catalogModel:     "claude-sonnet-4-6",
			providerPath:     false,
			wantAvailability: KnownCapabilityAvailability(false),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ModelGatedWebSearchAvailability(tt.provider, tt.model, tt.catalogModel, tt.providerPath)
			if got != tt.wantAvailability {
				t.Fatalf("ModelGatedWebSearchAvailability() = %+v, want %+v", got, tt.wantAvailability)
			}
		})
	}
}

func TestIsProviderCatalogModelListedUsesExactProviderCatalogOnly(t *testing.T) {
	if !IsProviderCatalogModelListed("gemini", "gemini-3.1-pro-preview-customtools") {
		t.Fatal("IsProviderCatalogModelListed(gemini exact) = false, want true")
	}
	if IsProviderCatalogModelListed("gemini", "gemini-3.1-pro-prod") {
		t.Fatal("IsProviderCatalogModelListed(gemini prefix alias) = true, want false")
	}
	if IsProviderCatalogModelListed("gemini", "gemini-pro") {
		t.Fatal("IsProviderCatalogModelListed(gemini-pro) = true, want false for legacy pricing-only metadata")
	}
}
