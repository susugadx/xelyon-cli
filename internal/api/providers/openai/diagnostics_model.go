package openai

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/cost"
	"github.com/susugadx/xelyon-cli/internal/llmcatalog"
)

func resolveOpenAIDiagnosticModel(cfg *config.Config, explicitModel string) (string, string) {
	if model := strings.TrimSpace(explicitModel); model != "" {
		return model, "--model"
	}
	if model := strings.TrimSpace(os.Getenv("XELYON_MODEL")); model != "" {
		return model, "XELYON_MODEL"
	}
	if model := strings.TrimSpace(cfg.GetExplicitProviderDefaultModel("openai")); model != "" {
		return model, "provider_models.openai.default_model"
	}
	if config.SameProviderRuntimeIdentity("openai", cfg.DefaultProvider) && strings.TrimSpace(cfg.DefaultModel) != "" {
		selected := strings.TrimSpace(cfg.GetSelectedModelForProvider("openai"))
		if selected == strings.TrimSpace(cfg.DefaultModel) {
			return selected, "default_model"
		}
	}
	if model := strings.TrimSpace(cfg.GetSelectedModelForProvider("openai")); model != "" {
		return model, "built-in provider default"
	}
	return "gpt-4o", "fallback"
}

func resolveOpenAIDiagnosticCatalogModel(cfg *config.Config, model, explicitCatalogModel string) (string, string) {
	model = strings.TrimSpace(model)
	if catalogModel := strings.TrimSpace(explicitCatalogModel); catalogModel != "" {
		return catalogModel, "--catalog-model"
	}
	if model == "" {
		return "", ""
	}
	if override, ok := cfg.ModelOverrideForProvider("openai", model); ok {
		if catalogModel := strings.TrimSpace(override.CatalogModel); catalogModel != "" {
			return catalogModel, "provider_models.openai.model_overrides"
		}
	}
	if pm, ok := cfg.GetProviderModelConfig("openai"); ok && strings.TrimSpace(pm.DefaultModel) == model {
		if catalogModel := strings.TrimSpace(pm.CatalogModel); catalogModel != "" {
			return catalogModel, "provider_models.openai.catalog_model"
		}
	}

	resolution := cfg.ResolveModelCatalog("openai", model)
	if strings.TrimSpace(resolution.Model) == "" {
		return model, "model"
	}
	if resolution.Model != model {
		return resolution.Model, "provider_models.openai.catalog_model"
	}
	if resolution.ConfiguredWithoutCatalog {
		return resolution.Model, "configured model"
	}
	return resolution.Model, "model"
}

type openAIDiagnosticRouteResolution struct {
	Route  string
	Reason string
}

func resolveOpenAIDiagnosticRouteResolution(cfg *config.Config, model, catalogModel string) openAIDiagnosticRouteResolution {
	model = strings.TrimSpace(model)
	catalogModel = strings.TrimSpace(catalogModel)
	if model == "" {
		return openAIDiagnosticRouteResolution{
			Reason: "model is not resolved",
		}
	}
	if cfg.IsProviderResponsesAPIModel("openai", model) {
		if ShouldStreamResponses(catalogModel) {
			return openAIDiagnosticRouteResolution{
				Route:  DiagnosticRouteResponsesStreaming,
				Reason: fmt.Sprintf("model=%s uses Responses API; %s", model, openAIDiagnosticResponsesStreamingReason(catalogModel, true)),
			}
		}
		return openAIDiagnosticRouteResolution{
			Route:  DiagnosticRouteResponsesNonStreaming,
			Reason: fmt.Sprintf("model=%s uses Responses API; %s", model, openAIDiagnosticResponsesStreamingReason(catalogModel, false)),
		}
	}
	return openAIDiagnosticRouteResolution{
		Route:  DiagnosticRouteChatCompletions,
		Reason: fmt.Sprintf("model=%s is not configured for Responses API", model),
	}
}

func openAIDiagnosticResponsesStreamingReason(catalogModel string, streaming bool) string {
	catalogModel = strings.TrimSpace(catalogModel)
	if catalogModel == "" {
		return "catalog_model is not resolved; Responses streaming defaults to enabled"
	}
	if streaming {
		return fmt.Sprintf("catalog_model=%s supports Responses streaming", catalogModel)
	}
	return fmt.Sprintf("catalog_model=%s disables Responses streaming", catalogModel)
}

func openAIDiagnosticPolicyConfig(cfg *config.Config, model, catalogModel string) *config.Config {
	return openAIDiagnosticConfigWithModelPolicy(cfg, model, catalogModel, 0)
}

func openAIDiagnosticConfigWithModelPolicy(cfg *config.Config, model, catalogModel string, maxOutputTokens int) *config.Config {
	policyCfg := config.CloneConfig(cfg)
	model = strings.TrimSpace(model)
	catalogModel = strings.TrimSpace(catalogModel)
	if model == "" || catalogModel == "" {
		return policyCfg
	}

	_ = policyCfg.PatchProviderModelConfig("openai", func(pm *config.ProviderModelConfig) {
		pm.DefaultModel = model
		pm.CatalogModel = catalogModel
		if pm.ModelOverrides == nil {
			pm.ModelOverrides = map[string]config.ModelOverride{}
		}
		override := pm.ModelOverrides[model]
		if existingOverride, ok := policyCfg.ModelOverrideForProvider("openai", model); ok {
			override = existingOverride
		}
		override.CatalogModel = catalogModel
		if maxOutputTokens > 0 {
			override.MaxOutputTokens = maxOutputTokens
		}
		pm.ModelOverrides[model] = override
	})
	return policyCfg
}

type openAIDiagnosticMaxOutputPolicyResult struct {
	Tokens    int
	Source    string
	Available bool
}

func openAIDiagnosticMaxOutputPolicy(cfg *config.Config, model, catalogModel string) openAIDiagnosticMaxOutputPolicyResult {
	if override, ok := cfg.ModelOverrideForProvider("openai", model); ok && override.MaxOutputTokens > 0 {
		return openAIDiagnosticMaxOutputPolicyResult{
			Tokens:    override.MaxOutputTokens,
			Source:    "model_overrides",
			Available: true,
		}
	}
	if tokens, ok := llmcatalog.KnownMaxOutputTokens(catalogModel); ok {
		return openAIDiagnosticMaxOutputPolicyResult{
			Tokens:    tokens,
			Source:    "catalog",
			Available: true,
		}
	}
	ctx := config.WithContext(context.Background(), cfg)
	tokens := api.GetMaxOutputTokens(ctx, "openai", model)
	if tokens <= 0 {
		return openAIDiagnosticMaxOutputPolicyResult{Source: "missing"}
	}
	return openAIDiagnosticMaxOutputPolicyResult{
		Tokens:    tokens,
		Source:    "provider_default",
		Available: true,
	}
}

func openAIDiagnosticMaxOutputTokens(cfg *config.Config, model, catalogModel string) (int, bool) {
	result := openAIDiagnosticMaxOutputPolicy(cfg, model, catalogModel)
	return result.Tokens, result.Available
}

func openAIDiagnosticIntDetail(value int, ok bool) string {
	if !ok {
		return "unknown"
	}
	return fmt.Sprintf("%d", value)
}

func openAIDiagnosticPricingDetail(pricing cost.PricingInfo) string {
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

func looksLikeOpenAICatalogModel(model string) bool {
	return llmcatalog.InferProviderFromModel(model) == "openai"
}
