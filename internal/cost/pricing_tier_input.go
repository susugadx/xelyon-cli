package cost

import (
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/llmcatalog"
)

func pricingTierInputTokensForUsage(cfg *config.Config, provider, model string, usage api.Usage) int {
	if cachedInputTokensAreInputDetails(cfg, provider, model) {
		return usage.InputTokens + usage.CacheCreationTokens
	}
	return usage.InputTokens + usage.CachedInputTokens + usage.CacheCreationTokens
}

func cachedInputTokensAreInputDetails(cfg *config.Config, provider, model string) bool {
	pricingModel := pricingModelForConfig(cfg, provider, model)
	entry, ok := llmcatalog.ProviderDescriptorFor(provider)
	if !ok {
		return false
	}

	switch entry.PricingFamily {
	case "openai":
		return true
	case "openrouter":
		return openRouterModelUsesOpenAIPricing(pricingModel)
	default:
		return false
	}
}

func openRouterModelUsesOpenAIPricing(model string) bool {
	lm := strings.ToLower(model)
	return containsAny(lm, []string{"gpt", "openai", "codex"})
}
