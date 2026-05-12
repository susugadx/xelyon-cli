package providerdiag

import (
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/llmcatalog"
)

// ResolveProviderDiagnosticModel は doctor 用の request model を provider 共通の優先順で解決する。
func ResolveProviderDiagnosticModel(cfg *config.Config, provider, explicitModel, fallbackModel string) (string, string) {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	provider = config.NormalizeProviderName(provider)

	if model := strings.TrimSpace(explicitModel); model != "" {
		return model, "--model"
	}
	if model := strings.TrimSpace(os.Getenv("XELYON_MODEL")); model != "" {
		return model, "XELYON_MODEL"
	}
	if model := strings.TrimSpace(cfg.GetExplicitProviderDefaultModel(provider)); model != "" {
		return model, "provider_models." + provider + ".default_model"
	}
	if config.SameProviderRuntimeIdentity(provider, cfg.DefaultProvider) && strings.TrimSpace(cfg.DefaultModel) != "" {
		selected := strings.TrimSpace(cfg.GetSelectedModelForProvider(provider))
		if selected == strings.TrimSpace(cfg.DefaultModel) {
			return selected, "default_model"
		}
	}
	if model := strings.TrimSpace(cfg.GetSelectedModelForProvider(provider)); model != "" {
		return model, "built-in provider default"
	}
	return strings.TrimSpace(fallbackModel), "fallback"
}

// ResolveProviderDiagnosticCatalogModel は doctor 用の catalog model を provider 共通の優先順で解決する。
func ResolveProviderDiagnosticCatalogModel(cfg *config.Config, provider, model, explicitCatalogModel string) (string, string) {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	provider = config.NormalizeProviderName(provider)
	model = strings.TrimSpace(model)

	if catalogModel := strings.TrimSpace(explicitCatalogModel); catalogModel != "" {
		return catalogModel, "--catalog-model"
	}
	if model == "" {
		return "", ""
	}
	if override, ok := cfg.ModelOverrideForProvider(provider, model); ok {
		if catalogModel := strings.TrimSpace(override.CatalogModel); catalogModel != "" {
			return catalogModel, "provider_models." + provider + ".model_overrides"
		}
	}
	if pm, ok := cfg.GetProviderModelConfig(provider); ok && strings.TrimSpace(pm.DefaultModel) == model {
		if catalogModel := strings.TrimSpace(pm.CatalogModel); catalogModel != "" {
			return catalogModel, "provider_models." + provider + ".catalog_model"
		}
	}

	resolution := cfg.ResolveModelCatalog(provider, model)
	if strings.TrimSpace(resolution.Model) == "" {
		return model, "model"
	}
	if resolution.Model != model {
		return resolution.Model, "provider_models." + provider + ".catalog_model"
	}
	if resolution.ConfiguredWithoutCatalog {
		return resolution.Model, "configured model"
	}
	return resolution.Model, "model"
}

// ProviderDiagnosticPolicyConfigOptions は doctor policy 用 config patch の入力を表す。
type ProviderDiagnosticPolicyConfigOptions struct {
	Provider        string
	Model           string
	CatalogModel    string
	MaxOutputTokens int
}

// ProviderDiagnosticPolicyConfig は provider doctor が runtime policy と同じ model/catalog/max token 解決を使うための config を返す。
func ProviderDiagnosticPolicyConfig(cfg *config.Config, options ProviderDiagnosticPolicyConfigOptions) *config.Config {
	policyCfg := config.CloneConfig(cfg)
	provider := config.NormalizeProviderName(options.Provider)
	model := strings.TrimSpace(options.Model)
	catalogModel := strings.TrimSpace(options.CatalogModel)
	invalidCatalogModel := catalogModel != "" && !IsProviderCatalogModelKnown(provider, catalogModel)
	if invalidCatalogModel {
		catalogModel = ""
	}
	if provider == "" || model == "" || (catalogModel == "" && options.MaxOutputTokens <= 0 && !invalidCatalogModel) {
		return policyCfg
	}

	_ = policyCfg.PatchProviderModelConfig(provider, func(pm *config.ProviderModelConfig) {
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
		if existingOverride, ok := policyCfg.ModelOverrideForProvider(provider, model); ok {
			override = existingOverride
		}
		if catalogModel != "" {
			override.CatalogModel = catalogModel
		} else if invalidCatalogModel {
			override.CatalogModel = ""
		}
		if options.MaxOutputTokens > 0 {
			override.MaxOutputTokens = options.MaxOutputTokens
		}
		pm.ModelOverrides[model] = override
	})
	return policyCfg
}

// IsProviderCatalogModelKnown は provider 所有の catalog ID / prefix だけを既知として扱う。
func IsProviderCatalogModelKnown(provider, model string) bool {
	return llmcatalog.IsKnownModelNameForProvider(provider, model)
}
