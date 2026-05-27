package deepseek

import (
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/llmcatalog"
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
)

func resolveDeepSeekDiagnosticModel(cfg *config.Config, explicitModel string) (string, string) {
	return providerdiag.ResolveProviderDiagnosticModel(cfg, "deepseek", explicitModel, llmcatalog.DefaultModelForProvider("deepseek"))
}

func resolveDeepSeekDiagnosticCatalogModel(cfg *config.Config, model, explicitCatalogModel string) (string, string) {
	return providerdiag.ResolveProviderDiagnosticCatalogModel(cfg, "deepseek", model, explicitCatalogModel)
}

func deepSeekDiagnosticPolicyConfig(cfg *config.Config, model, catalogModel string, maxOutputTokens int) *config.Config {
	return providerdiag.ProviderDiagnosticPolicyConfig(cfg, providerdiag.ProviderDiagnosticPolicyConfigOptions{
		Provider:        "deepseek",
		Model:           model,
		CatalogModel:    catalogModel,
		MaxOutputTokens: maxOutputTokens,
	})
}

func deepSeekCatalogModelKnown(model string) bool {
	return providerdiag.IsProviderCatalogModelKnown("deepseek", model)
}
