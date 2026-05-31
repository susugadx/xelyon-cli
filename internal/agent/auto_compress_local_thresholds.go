package agent

import (
	"context"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/llmcatalog"
	"github.com/susugadx/xelyon-cli/internal/token"
)

const minProviderFallbackInputBudgetTokens = 8192

func localAutoCompressionTokenThresholdForConfig(cfg *config.Config, provider, model string) (int, bool) {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	limit, ok := token.GetKnownModelTokenLimitForConfig(cfg, provider, model)
	if !ok || limit <= 0 {
		return 0, false
	}

	triggerPercent := cfg.Compression.TriggerPercent
	if triggerPercent == 0 {
		triggerPercent = 80
	}
	percentThreshold := limit * triggerPercent / 100
	if headroomThreshold := localAutoCompressionOutputHeadroomThreshold(cfg, provider, model, limit); headroomThreshold > 0 && headroomThreshold < percentThreshold {
		return headroomThreshold, true
	}
	return percentThreshold, true
}

func localAutoCompressionOutputHeadroomThreshold(cfg *config.Config, provider, model string, contextLimit int) int {
	if contextLimit <= 0 {
		return 0
	}

	maxOutputTokens, ok := localAutoCompressionRequestMaxOutputTokens(cfg, provider, model, contextLimit)
	if !ok || maxOutputTokens <= 0 {
		return 0
	}

	inputBudget := contextLimit - maxOutputTokens
	if inputBudget <= 0 {
		return 1
	}
	return inputBudget
}

func localAutoCompressionRequestMaxOutputTokens(cfg *config.Config, provider, model string, contextLimit int) (int, bool) {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	if _, ok := localAutoCompressionModelMaxOutputTokens(cfg, provider, model); ok {
		return api.GetMaxOutputTokens(config.WithContext(context.Background(), cfg), provider, model), true
	}
	if _, ok := localAutoCompressionProviderMaxOutputTokens(cfg, provider, model); !ok {
		return 0, false
	}

	maxOutputTokens := api.GetMaxOutputTokens(config.WithContext(context.Background(), cfg), provider, model)
	if !localAutoCompressionProviderFallbackLeavesUsableInputBudget(contextLimit, maxOutputTokens) {
		return 0, false
	}
	return maxOutputTokens, true
}

func localAutoCompressionModelMaxOutputTokens(cfg *config.Config, provider, model string) (int, bool) {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	if override, ok := cfg.ModelOverrideForProvider(provider, model); ok && override.MaxOutputTokens > 0 {
		return override.MaxOutputTokens, true
	}
	if tokens, ok := llmcatalog.KnownMaxOutputTokens(cfg.ModelCatalogName(provider, model)); ok {
		return tokens, true
	}
	return 0, false
}

func localAutoCompressionProviderMaxOutputTokens(cfg *config.Config, provider, model string) (int, bool) {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}
	lookupProvider := cfg.RuntimeProviderConfigKey(provider, model)
	if providerConfig, ok := cfg.GetProviderModelConfig(lookupProvider); ok && providerConfig.MaxOutputTokens > 0 {
		return providerConfig.MaxOutputTokens, true
	}
	return 0, false
}

func localAutoCompressionProviderFallbackLeavesUsableInputBudget(contextLimit, maxOutputTokens int) bool {
	if contextLimit <= 0 || maxOutputTokens <= 0 {
		return false
	}

	inputBudget := contextLimit - maxOutputTokens
	if inputBudget <= 0 {
		return false
	}

	minInputBudget := contextLimit / 10
	if minInputBudget < minProviderFallbackInputBudgetTokens {
		minInputBudget = minProviderFallbackInputBudgetTokens
	}
	return inputBudget >= minInputBudget
}
