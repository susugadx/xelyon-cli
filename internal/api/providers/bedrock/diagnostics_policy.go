package bedrock

import (
	"fmt"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/cost"
	"github.com/susugadx/xelyon-cli/internal/providerdiag"
)

func (r *DiagnosticReport) addCatalogPolicyCheck(cfg *config.Config, route bedrockRoute) {
	model := strings.TrimSpace(r.Model)
	catalogModel := strings.TrimSpace(r.CatalogModel)
	if model == "" || catalogModel == "" {
		return
	}

	policy := bedrockDiagnosticCatalogPolicy(cfg, route, model, catalogModel)
	detail := bedrockDiagnosticCatalogPolicyDetail(policy)

	switch {
	case !policy.ContextWindowKnown:
		r.addCheck(DiagnosticStatusWarn, "catalog_policy", "catalog_model is missing context window metadata", detail, "Use a Bedrock model ID known to XELYON before relying on token-limit diagnostics")
	case !policy.MaxOutput.Available:
		r.addCheck(DiagnosticStatusWarn, "catalog_policy", "catalog_model is missing max output metadata", detail, "Converse requests will omit maxTokens unless a known catalog model or max_output_tokens override is configured")
	case policy.Pricing.PricingUnavailable:
		r.addCheck(DiagnosticStatusWarn, "catalog_policy", "catalog_model is missing Bedrock pricing metadata", detail, "Use a Bedrock model ID with pricing metadata before relying on cost estimates")
	default:
		r.addCheck(DiagnosticStatusOK, "catalog_policy", "catalog_model policy is available", detail, "")
	}
}

func bedrockDiagnosticCatalogPolicy(cfg *config.Config, route bedrockRoute, model, catalogModel string) providerdiag.CatalogPolicy {
	model = strings.TrimSpace(model)
	catalogModel = strings.TrimSpace(catalogModel)
	policyCfg := bedrockDiagnosticPolicyConfig(cfg, model, catalogModel)
	maxOutput := bedrockDiagnosticMaxOutputPolicy(policyCfg, route, model, catalogModel)
	return providerdiag.BedrockCatalogPolicy(policyCfg, model, catalogModel, maxOutput)
}

func bedrockDiagnosticCatalogPolicyDetail(policy providerdiag.CatalogPolicy) string {
	return fmt.Sprintf(
		"catalog_model=%s, context_window=%s, max_output_tokens=%s, %s",
		policy.CatalogModel,
		bedrockDiagnosticIntDetail(policy.ContextWindowTokens, policy.ContextWindowKnown),
		bedrockDiagnosticIntDetail(policy.MaxOutput.Tokens, policy.MaxOutput.Available),
		bedrockDiagnosticPricingDetail(policy.Pricing),
	)
}

func bedrockDiagnosticPolicyConfig(cfg *config.Config, model, catalogModel string) *config.Config {
	policyCfg := config.CloneConfig(cfg)
	providerCfg, _ := policyCfg.GetProviderModelConfig("bedrock")
	override := config.ModelOverride{CatalogModel: catalogModel}
	if existingOverride, ok := policyCfg.ModelOverrideForProvider("bedrock", model); ok {
		override = existingOverride
		override.CatalogModel = catalogModel
	}
	providerCfg.DefaultModel = model
	providerCfg.CatalogModel = catalogModel
	if providerCfg.ModelOverrides == nil {
		providerCfg.ModelOverrides = map[string]config.ModelOverride{}
	}
	providerCfg.ModelOverrides[model] = override
	policyCfg.SetProviderModelConfig("bedrock", providerCfg)
	return policyCfg
}

func bedrockDiagnosticMaxOutputPolicy(cfg *config.Config, route bedrockRoute, model, catalogModel string) providerdiag.MaxOutputPolicy {
	if route == bedrockRouteConverseStream {
		return bedrockDiagnosticProviderMaxOutputPolicy(converseMaxOutputPolicy(bedrockRequestContext{
			model:        model,
			catalogModel: catalogModel,
			route:        route,
			cfg:          cfg,
		}))
	}
	providerCfg, _ := cfg.GetProviderModelConfig("bedrock")
	return bedrockDiagnosticProviderMaxOutputPolicy(claudeMessagesMaxOutputPolicy(bedrockRequestContext{
		model:          model,
		catalogModel:   catalogModel,
		route:          route,
		cfg:            cfg,
		providerConfig: providerCfg,
	}))
}

func bedrockDiagnosticProviderMaxOutputPolicy(policy bedrockMaxOutputPolicy) providerdiag.MaxOutputPolicy {
	return providerdiag.MaxOutputPolicy{
		Tokens:    policy.tokens,
		Source:    bedrockDiagnosticMaxOutputSource(policy.source),
		Available: policy.available,
	}
}

func bedrockDiagnosticMaxOutputSource(source bedrockMaxOutputSource) string {
	switch source {
	case bedrockMaxOutputSourceModelOverrides:
		return providerdiag.MaxOutputSourceModelOverrides
	case bedrockMaxOutputSourceCatalog:
		return providerdiag.MaxOutputSourceCatalog
	case bedrockMaxOutputSourceProviderDefault:
		return providerdiag.MaxOutputSourceProviderDefault
	default:
		return providerdiag.MaxOutputSourceMissing
	}
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
