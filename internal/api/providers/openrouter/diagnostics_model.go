package openrouter

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/llmcatalog"
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
)

type openRouterDiagnosticCatalogModelUse struct {
	CatalogKnown       bool
	Trusted            bool
	PolicyCatalogModel string
	MismatchDetail     string
	MismatchSuggestion string
}

func resolveOpenRouterDiagnosticModel(cfg *config.Config, explicitModel string) (string, string) {
	return providerdiag.ResolveProviderDiagnosticModel(cfg, "openrouter", explicitModel, llmcatalog.DefaultModelForProvider("openrouter"))
}

func resolveOpenRouterDiagnosticCatalogModel(cfg *config.Config, model, explicitCatalogModel string) (string, string) {
	return providerdiag.ResolveProviderDiagnosticCatalogModel(cfg, "openrouter", model, explicitCatalogModel)
}

func openRouterDiagnosticPolicyConfig(cfg *config.Config, model, catalogModel string, maxOutputTokens int) *config.Config {
	catalogUse := resolveOpenRouterDiagnosticCatalogModelUse(model, catalogModel)
	if strings.TrimSpace(catalogUse.PolicyCatalogModel) == "" {
		return openRouterDiagnosticPolicyConfigWithoutCatalog(cfg, model, maxOutputTokens)
	}
	return providerdiag.ProviderDiagnosticPolicyConfig(cfg, providerdiag.ProviderDiagnosticPolicyConfigOptions{
		Provider:        "openrouter",
		Model:           model,
		CatalogModel:    catalogUse.PolicyCatalogModel,
		MaxOutputTokens: maxOutputTokens,
	})
}

func openRouterDiagnosticPolicyConfigWithoutCatalog(cfg *config.Config, model string, maxOutputTokens int) *config.Config {
	policyCfg := config.CloneConfig(cfg)
	model = strings.TrimSpace(model)
	if model == "" {
		return policyCfg
	}

	_ = policyCfg.PatchProviderModelConfig("openrouter", func(pm *config.ProviderModelConfig) {
		pm.DefaultModel = model
		pm.CatalogModel = ""
		if pm.ModelOverrides == nil {
			pm.ModelOverrides = map[string]config.ModelOverride{}
		}
		override := pm.ModelOverrides[model]
		if existingOverride, ok := policyCfg.ModelOverrideForProvider("openrouter", model); ok {
			override = existingOverride
		}
		override.CatalogModel = ""
		if maxOutputTokens > 0 {
			override.MaxOutputTokens = maxOutputTokens
		}
		pm.ModelOverrides[model] = override
	})
	return policyCfg
}

func openRouterCatalogModelKnown(model string) bool {
	return providerdiag.IsProviderCatalogModelKnown("openrouter", model)
}

func resolveOpenRouterDiagnosticCatalogModelUse(model, catalogModel string) openRouterDiagnosticCatalogModelUse {
	model = strings.TrimSpace(model)
	catalogModel = strings.TrimSpace(catalogModel)
	use := openRouterDiagnosticCatalogModelUse{
		CatalogKnown:       openRouterCatalogModelKnown(catalogModel),
		Trusted:            true,
		PolicyCatalogModel: catalogModel,
	}
	if !use.CatalogKnown {
		use.Trusted = false
		use.PolicyCatalogModel = ""
		return use
	}
	if strings.EqualFold(model, catalogModel) {
		return use
	}

	modelOwner, _, modelRouted := splitOpenRouterModelID(model)
	catalogOwner, _, catalogRouted := splitOpenRouterModelID(catalogModel)
	if openRouterCatalogModelKnown(model) {
		use.Trusted = false
		use.PolicyCatalogModel = model
		use.MismatchDetail = fmt.Sprintf("model=%s is a known OpenRouter routed model; catalog_model=%s describes a different model", model, catalogModel)
		use.MismatchSuggestion = "Use the request model itself as catalog_model, or remove --catalog-model for direct OpenRouter model IDs"
		return use
	}
	if modelRouted && catalogRouted && !strings.EqualFold(modelOwner, catalogOwner) {
		use.Trusted = false
		use.PolicyCatalogModel = ""
		use.MismatchDetail = fmt.Sprintf("model owner=%s but catalog_model owner=%s", modelOwner, catalogOwner)
		use.MismatchSuggestion = "Use a catalog_model with the same OpenRouter owner prefix, or use an unprefixed alias model when mapping to a different provider family"
		return use
	}
	return use
}

func openRouterDiagnosticPolicyCatalogModel(model, catalogModel string) string {
	return resolveOpenRouterDiagnosticCatalogModelUse(model, catalogModel).PolicyCatalogModel
}

func resolveOpenRouterDiagnosticUpstreamModel(model, catalogModel string) (string, string) {
	candidate := strings.TrimSpace(model)
	if openRouterCatalogModelKnown(catalogModel) {
		candidate = strings.TrimSpace(catalogModel)
	}

	owner, routed, ok := splitOpenRouterModelID(candidate)
	if !ok {
		return "", candidate
	}
	return owner, routed
}

func splitOpenRouterModelID(model string) (string, string, bool) {
	parts := strings.SplitN(strings.TrimSpace(model), "/", 2)
	if len(parts) != 2 || strings.TrimSpace(parts[0]) == "" || strings.TrimSpace(parts[1]) == "" {
		return "", "", false
	}
	return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]), true
}
