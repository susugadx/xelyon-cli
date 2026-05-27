package bedrock

import (
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/llmcatalog"
)

func resolveBedrockDiagnosticModel(cfg *config.Config, explicitModel string) (string, string) {
	if model := strings.TrimSpace(explicitModel); model != "" {
		return model, "--model"
	}
	if envModel := strings.TrimSpace(os.Getenv("XELYON_MODEL")); envModel != "" {
		return envModel, "XELYON_MODEL"
	}
	if explicit := strings.TrimSpace(cfg.GetExplicitProviderDefaultModel("bedrock")); explicit != "" {
		return explicit, "provider_models.bedrock.default_model"
	}
	if config.SameProviderRuntimeIdentity("bedrock", cfg.DefaultProvider) && strings.TrimSpace(cfg.DefaultModel) != "" {
		selected := strings.TrimSpace(cfg.GetSelectedModelForProvider("bedrock"))
		if selected == strings.TrimSpace(cfg.DefaultModel) {
			return selected, "default_model"
		}
	}
	if selected := strings.TrimSpace(cfg.GetEffectiveModelForProvider("bedrock")); selected != "" {
		return selected, "built-in provider default"
	}
	return llmcatalog.DefaultModelForProvider("bedrock"), "provider fallback"
}

func resolveBedrockDiagnosticCatalogModel(cfg *config.Config, model, explicitCatalogModel string) (string, string) {
	model = strings.TrimSpace(model)
	if catalogModel := strings.TrimSpace(explicitCatalogModel); catalogModel != "" {
		return catalogModel, "--catalog-model"
	}
	if model == "" {
		return "", ""
	}

	if override, ok := cfg.ModelOverrideForProvider("bedrock", model); ok {
		if catalogModel := strings.TrimSpace(override.CatalogModel); catalogModel != "" {
			return catalogModel, "provider_models.bedrock.model_overrides"
		}
	}

	if pm, ok := cfg.GetProviderModelConfig("bedrock"); ok && strings.TrimSpace(pm.DefaultModel) == model {
		if catalogModel := strings.TrimSpace(pm.CatalogModel); catalogModel != "" {
			return catalogModel, "provider_models.bedrock.catalog_model"
		}
	}

	return cfg.ModelCatalogName("bedrock", model), "model name fallback"
}
