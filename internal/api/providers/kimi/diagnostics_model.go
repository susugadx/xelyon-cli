package kimi

import (
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/llmcatalog"
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
)

func resolveKimiDiagnosticModel(cfg *config.Config, explicitModel string) (string, string) {
	if model := strings.TrimSpace(explicitModel); model != "" {
		return model, "--model"
	}
	if model := strings.TrimSpace(os.Getenv("XELYON_MODEL")); model != "" {
		return model, "XELYON_MODEL"
	}
	if model := strings.TrimSpace(cfg.GetExplicitProviderDefaultModel("kimi")); model != "" {
		return model, "provider_models.kimi.default_model"
	}
	if config.SameProviderRuntimeIdentity("kimi", cfg.DefaultProvider) && strings.TrimSpace(cfg.DefaultModel) != "" {
		selected := strings.TrimSpace(cfg.GetSelectedModelForProvider("kimi"))
		if selected == strings.TrimSpace(cfg.DefaultModel) {
			return selected, "default_model"
		}
	}
	if model := strings.TrimSpace(cfg.GetSelectedModelForProvider("kimi")); model != "" {
		return model, "built-in provider default"
	}
	return llmcatalog.DefaultModelForProvider("kimi"), "fallback"
}

func resolveKimiDiagnosticCatalogModel(cfg *config.Config, model, explicitCatalogModel string) (string, string) {
	return providerdiag.ResolveProviderDiagnosticCatalogModel(cfg, "kimi", model, explicitCatalogModel)
}

func kimiDiagnosticPolicyConfig(cfg *config.Config, model, catalogModel string, maxOutputTokens int) *config.Config {
	return providerdiag.ProviderDiagnosticPolicyConfig(cfg, providerdiag.ProviderDiagnosticPolicyConfigOptions{
		Provider:        "kimi",
		Model:           model,
		CatalogModel:    catalogModel,
		MaxOutputTokens: maxOutputTokens,
	})
}
