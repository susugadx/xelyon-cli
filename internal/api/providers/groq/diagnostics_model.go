package groq

import (
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/llmcatalog"
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
)

func resolveGroqDiagnosticModel(cfg *config.Config, explicitModel string) (string, string) {
	return providerdiag.ResolveProviderDiagnosticModel(cfg, "groq", explicitModel, llmcatalog.DefaultModelForProvider("groq"))
}

func resolveGroqDiagnosticCatalogModel(cfg *config.Config, model, explicitCatalogModel string) (string, string) {
	return providerdiag.ResolveProviderDiagnosticCatalogModel(cfg, "groq", model, explicitCatalogModel)
}

func groqDiagnosticPolicyConfig(cfg *config.Config, model, catalogModel string, maxOutputTokens int) *config.Config {
	return providerdiag.ProviderDiagnosticPolicyConfig(cfg, providerdiag.ProviderDiagnosticPolicyConfigOptions{
		Provider:        "groq",
		Model:           model,
		CatalogModel:    catalogModel,
		MaxOutputTokens: maxOutputTokens,
	})
}

func groqCatalogModelKnown(model string) bool {
	return providerdiag.IsProviderCatalogModelKnown("groq", model)
}
