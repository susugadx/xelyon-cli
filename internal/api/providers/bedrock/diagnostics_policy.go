package bedrock

import (
	"context"
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/cost"
	"github.com/susugadx/xelyon-cli/internal/llmcatalog"
)

func (r *DiagnosticReport) addCatalogPolicyCheck(cfg *config.Config, route bedrockRoute) {
	model := strings.TrimSpace(r.Model)
	catalogModel := strings.TrimSpace(r.CatalogModel)
	if model == "" || catalogModel == "" {
		return
	}

	policyCfg := bedrockDiagnosticPolicyConfig(cfg, model, catalogModel)
	contextWindow, contextOK := llmcatalog.KnownModelContextLimit(catalogModel)
	maxOutput, maxOutputOK := bedrockDiagnosticMaxOutputTokens(policyCfg, route, model, catalogModel)
	pricing := cost.GetPricingInfoForConfig(policyCfg, "bedrock", model)
	detail := fmt.Sprintf(
		"catalog_model=%s, context_window=%s, max_output_tokens=%s, %s",
		catalogModel,
		bedrockDiagnosticIntDetail(contextWindow, contextOK),
		bedrockDiagnosticIntDetail(maxOutput, maxOutputOK),
		bedrockDiagnosticPricingDetail(pricing),
	)

	switch {
	case !contextOK:
		r.addCheck(DiagnosticStatusWarn, "catalog_policy", "catalog_model is missing context window metadata", detail, "Use a Bedrock model ID known to XELYON before relying on token-limit diagnostics")
	case !maxOutputOK:
		r.addCheck(DiagnosticStatusWarn, "catalog_policy", "catalog_model is missing max output metadata", detail, "Converse requests will omit maxTokens unless a known catalog model or max_output_tokens override is configured")
	case pricing.PricingUnavailable:
		r.addCheck(DiagnosticStatusWarn, "catalog_policy", "catalog_model is missing Bedrock pricing metadata", detail, "Use a Bedrock model ID with pricing metadata before relying on cost estimates")
	default:
		r.addCheck(DiagnosticStatusOK, "catalog_policy", "catalog_model policy is available", detail, "")
	}
}

func bedrockDiagnosticPolicyConfig(cfg *config.Config, model, catalogModel string) *config.Config {
	policyCfg := config.CloneConfig(cfg)
	override := config.ModelOverride{CatalogModel: catalogModel}
	if existingOverride, ok := policyCfg.ModelOverrideForProvider("bedrock", model); ok {
		override = existingOverride
		override.CatalogModel = catalogModel
	}
	policyCfg.SetProviderModelConfig("bedrock", config.ProviderModelConfig{
		DefaultModel: model,
		CatalogModel: catalogModel,
		ModelOverrides: map[string]config.ModelOverride{
			model: override,
		},
	})
	return policyCfg
}

func bedrockDiagnosticMaxOutputTokens(cfg *config.Config, route bedrockRoute, model, catalogModel string) (int, bool) {
	if route == bedrockRouteConverseStream {
		return converseMaxTokens(bedrockRequestContext{
			model:        model,
			catalogModel: catalogModel,
			route:        route,
			cfg:          cfg,
		})
	}
	ctx := config.WithContext(context.Background(), cfg)
	maxTokens := api.GetMaxOutputTokens(ctx, "bedrock", model)
	return maxTokens, maxTokens > 0
}

func bedrockDiagnosticIntDetail(value int, ok bool) string {
	if !ok {
		return "unknown"
	}
	return fmt.Sprintf("%d", value)
}

func bedrockDiagnosticPricingDetail(pricing cost.PricingInfo) string {
	if pricing.PricingUnavailable {
		return "pricing=unavailable"
	}
	return fmt.Sprintf(
		"pricing=input $%.2f/M cached $%.3f/M output $%.2f/M",
		pricing.InputCostPerM,
		pricing.CachedInputCostPerM,
		pricing.OutputCostPerM,
	)
}
