package ollama

import (
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
)

const ollamaBaseURLEnv = "OLLAMA_BASE_URL"

func resolveOllamaDiagnosticModel(cfg *config.Config, explicitModel string) (string, string) {
	return providerdiag.ResolveProviderDiagnosticModel(cfg, "ollama", explicitModel, defaultOllamaModel)
}

func resolveOllamaDiagnosticCatalogModel(cfg *config.Config, model, explicitCatalogModel string) (string, string) {
	return providerdiag.ResolveProviderDiagnosticCatalogModel(cfg, "ollama", model, explicitCatalogModel)
}

func ollamaDiagnosticPolicyConfig(cfg *config.Config, model, catalogModel string, maxOutputTokens int) *config.Config {
	return providerdiag.ProviderDiagnosticPolicyConfig(cfg, providerdiag.ProviderDiagnosticPolicyConfigOptions{
		Provider:        "ollama",
		Model:           model,
		CatalogModel:    catalogModel,
		MaxOutputTokens: maxOutputTokens,
	})
}

func ollamaCatalogModelKnown(model string) bool {
	return providerdiag.IsProviderCatalogModelKnown("ollama", model)
}

func resolveOllamaDiagnosticBaseURL() string {
	if raw := strings.TrimSpace(os.Getenv(ollamaBaseURLEnv)); raw != "" {
		return raw
	}
	return New("").BaseURL()
}
