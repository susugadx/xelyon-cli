package kimi

import (
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
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
	return defaultKimiModel, "fallback"
}

func resolveKimiDiagnosticCatalogModel(cfg *config.Config, model string) (string, string) {
	model = strings.TrimSpace(model)
	if model == "" {
		return "", ""
	}
	resolution := cfg.ResolveModelCatalog("kimi", model)
	if strings.TrimSpace(resolution.Model) == "" {
		return model, "model"
	}
	if resolution.Model != model {
		return resolution.Model, "provider_models.kimi.catalog_model"
	}
	if resolution.ConfiguredWithoutCatalog {
		return resolution.Model, "configured model"
	}
	return resolution.Model, "model"
}
