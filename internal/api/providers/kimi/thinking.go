package kimi

import (
	"context"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func kimiThinkingConfig(ctx context.Context, providerConfigKey, requestedModel string) (map[string]any, bool, string) {
	return kimiThinkingConfigForResolved(resolveKimiRequestOptions(ctx, providerConfigKey, requestedModel, ""))
}

func kimiThinkingConfigForResolved(options kimiResolvedRequestOptions) (map[string]any, bool, string) {
	if isKimiForcedThinkingModel(options.requestedModel) || isKimiForcedThinkingModel(options.catalogModel) {
		if api.IsThinkingEnabled(options.ctx) {
			return kimiThinkingEnabledField(kimiThinkingPayloadModel(options.requestedModel, options.catalogModel)), true, "Reasoner"
		}
		return nil, true, "Reasoner"
	}
	if !isKimiConfigurableThinkingModel(options.requestedModel) && !isKimiConfigurableThinkingModel(options.catalogModel) {
		return nil, false, ""
	}
	if api.IsThinkingEnabled(options.ctx) {
		return kimiThinkingEnabledField(kimiThinkingPayloadModel(options.requestedModel, options.catalogModel)), true, "Reasoner"
	}
	return map[string]any{
		"thinking": map[string]any{"type": "disabled"},
	}, false, ""
}

func kimiThinkingEnabledField(model string) map[string]any {
	thinking := map[string]any{"type": "enabled"}
	if isKimiKeepAllThinkingModel(model) {
		thinking["keep"] = "all"
	}
	return map[string]any{
		"thinking": thinking,
	}
}

func kimiCatalogModel(ctx context.Context, providerConfigKey, requestedModel string) string {
	catalogModel := config.FromContext(ctx).ModelCatalogName(providerConfigKey, requestedModel)
	if strings.TrimSpace(catalogModel) == "" {
		return requestedModel
	}
	return catalogModel
}

func kimiThinkingPayloadModel(requestedModel, catalogModel string) string {
	if isKimiConfigurableThinkingModel(catalogModel) || isKimiForcedThinkingModel(catalogModel) {
		return catalogModel
	}
	return requestedModel
}

func kimiWebSearchThinkingField(options kimiResolvedRequestOptions) map[string]any {
	if isKimiForcedThinkingModel(options.requestedModel) || isKimiForcedThinkingModel(options.catalogModel) {
		return nil
	}
	return map[string]any{"type": "disabled"}
}

func isKimiConfigurableThinkingModel(model string) bool {
	switch strings.ToLower(strings.TrimSpace(model)) {
	case "kimi-k2.6", "kimi-k2.5":
		return true
	default:
		return false
	}
}

func isKimiForcedThinkingModel(model string) bool {
	return strings.EqualFold(strings.TrimSpace(model), "kimi-k2-thinking")
}

func isKimiKeepAllThinkingModel(model string) bool {
	return strings.EqualFold(strings.TrimSpace(model), "kimi-k2.6")
}
