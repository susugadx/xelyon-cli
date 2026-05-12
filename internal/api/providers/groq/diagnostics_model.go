package groq

import (
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/llmcatalog"
)

func resolveGroqDiagnosticModel(cfg *config.Config, explicitModel string) (string, string) {
	if model := strings.TrimSpace(explicitModel); model != "" {
		return model, "--model"
	}
	if model := strings.TrimSpace(os.Getenv("XELYON_MODEL")); model != "" {
		return model, "XELYON_MODEL"
	}
	if model := strings.TrimSpace(cfg.GetExplicitProviderDefaultModel("groq")); model != "" {
		return model, "provider_models.groq.default_model"
	}
	if config.SameProviderRuntimeIdentity("groq", cfg.DefaultProvider) && strings.TrimSpace(cfg.DefaultModel) != "" {
		selected := strings.TrimSpace(cfg.GetSelectedModelForProvider("groq"))
		if selected == strings.TrimSpace(cfg.DefaultModel) {
			return selected, "default_model"
		}
	}
	if model := strings.TrimSpace(cfg.GetSelectedModelForProvider("groq")); model != "" {
		return model, "built-in provider default"
	}
	return "meta-llama/llama-4-scout-17b-16e-instruct", "fallback"
}

func resolveGroqDiagnosticCatalogModel(cfg *config.Config, model, explicitCatalogModel string) (string, string) {
	model = strings.TrimSpace(model)
	if catalogModel := strings.TrimSpace(explicitCatalogModel); catalogModel != "" {
		return catalogModel, "--catalog-model"
	}
	if model == "" {
		return "", ""
	}
	if override, ok := cfg.ModelOverrideForProvider("groq", model); ok {
		if catalogModel := strings.TrimSpace(override.CatalogModel); catalogModel != "" {
			return catalogModel, "provider_models.groq.model_overrides"
		}
	}
	if pm, ok := cfg.GetProviderModelConfig("groq"); ok && strings.TrimSpace(pm.DefaultModel) == model {
		if catalogModel := strings.TrimSpace(pm.CatalogModel); catalogModel != "" {
			return catalogModel, "provider_models.groq.catalog_model"
		}
	}

	resolution := cfg.ResolveModelCatalog("groq", model)
	if strings.TrimSpace(resolution.Model) == "" {
		return model, "model"
	}
	if resolution.Model != model {
		return resolution.Model, "provider_models.groq.catalog_model"
	}
	if resolution.ConfiguredWithoutCatalog {
		return resolution.Model, "configured model"
	}
	return resolution.Model, "model"
}

func groqDiagnosticPolicyConfig(cfg *config.Config, model, catalogModel string, maxOutputTokens int) *config.Config {
	policyCfg := config.CloneConfig(cfg)
	model = strings.TrimSpace(model)
	catalogModel = strings.TrimSpace(catalogModel)
	invalidCatalogModel := catalogModel != "" && !groqCatalogModelKnown(catalogModel)
	if invalidCatalogModel {
		catalogModel = ""
	}
	if model == "" || (catalogModel == "" && maxOutputTokens <= 0 && !invalidCatalogModel) {
		return policyCfg
	}

	_ = policyCfg.PatchProviderModelConfig("groq", func(pm *config.ProviderModelConfig) {
		pm.DefaultModel = model
		if catalogModel != "" {
			pm.CatalogModel = catalogModel
		} else if invalidCatalogModel {
			pm.CatalogModel = ""
		}
		if pm.ModelOverrides == nil {
			pm.ModelOverrides = map[string]config.ModelOverride{}
		}
		override := pm.ModelOverrides[model]
		if existingOverride, ok := policyCfg.ModelOverrideForProvider("groq", model); ok {
			override = existingOverride
		}
		if catalogModel != "" {
			override.CatalogModel = catalogModel
		} else if invalidCatalogModel {
			override.CatalogModel = ""
		}
		if maxOutputTokens > 0 {
			override.MaxOutputTokens = maxOutputTokens
		}
		pm.ModelOverrides[model] = override
	})
	return policyCfg
}

func groqCatalogModelKnown(model string) bool {
	return llmcatalog.IsKnownModelNameForProvider("groq", model)
}
