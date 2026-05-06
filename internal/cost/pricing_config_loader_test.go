package cost

import (
	"sync"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/llmcatalog"
)

func TestGetPricingInfo_FallbackToHardcodedWhenYAMLUnavailable(t *testing.T) {
	origEmbedded := embeddedPricingYAML
	origCfg := loadedPricingConfig

	embeddedPricingYAML = nil
	pricingConfigOnce = sync.Once{}
	loadedPricingConfig = nil
	t.Cleanup(func() {
		embeddedPricingYAML = origEmbedded
		pricingConfigOnce = sync.Once{}
		loadedPricingConfig = origCfg
	})

	pricing := GetPricingInfo("openai", "gpt-5")
	if pricing.InputCostPerM != 1.25 || pricing.OutputCostPerM != 10.00 {
		t.Fatalf("fallback pricing mismatch: got input=%f output=%f", pricing.InputCostPerM, pricing.OutputCostPerM)
	}
}

func TestPricingFamilyHasKnownModelUsesExactAllowlist(t *testing.T) {
	tests := []struct {
		name   string
		family string
		model  string
		want   bool
	}{
		{name: "openai exact", family: "openai", model: "gpt-5.3-codex", want: true},
		{name: "openai rule-only", family: "openai", model: "gpt-5.3", want: false},
		{name: "openai alias contains exact", family: "openai", model: "corp-gpt-5.3-codex-prod", want: false},
		{name: "bedrock exact", family: "bedrock", model: "global.anthropic.claude-sonnet-4-6", want: true},
		{name: "bedrock regional profile exact", family: "bedrock", model: "us.anthropic.claude-sonnet-4-6", want: true},
		{name: "bedrock eu profile exact", family: "bedrock", model: "eu.anthropic.claude-sonnet-4-6", want: true},
		{name: "bedrock au profile exact", family: "bedrock", model: "au.anthropic.claude-sonnet-4-6", want: true},
		{name: "bedrock legacy exact", family: "bedrock", model: "global.anthropic.claude-sonnet-4-6-v1", want: true},
		{name: "bedrock legacy versioned exact", family: "bedrock", model: "global.anthropic.claude-sonnet-4-6-v1:0", want: true},
		{name: "bedrock alias contains claude", family: "bedrock", model: "corp-claude-prod", want: false},
		{name: "kimi exact", family: "kimi", model: "kimi-k2.6", want: true},
		{name: "kimi thinking exact", family: "kimi", model: "kimi-k2-thinking", want: true},
		{name: "kimi alias contains exact", family: "kimi", model: "corp-kimi-k2.6-prod", want: false},
		{name: "openrouter exact", family: "openrouter", model: "openai/gpt-5.4", want: true},
		{name: "openrouter non-existent delegated id", family: "openrouter", model: "openai/gpt-5.3", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pricingFamilyHasKnownModel(tt.family, tt.model); got != tt.want {
				t.Fatalf("pricingFamilyHasKnownModel(%q, %q) = %v, want %v", tt.family, tt.model, got, tt.want)
			}
		})
	}
}

func TestKnownPricingModelsResolveToAvailablePricing(t *testing.T) {
	cfg := loadPricingConfig()
	if cfg == nil {
		t.Fatal("loadPricingConfig() = nil")
	}

	for family, provider := range *cfg {
		for _, model := range provider.KnownModels.Exact {
			t.Run(family+"/"+model, func(t *testing.T) {
				got := knownPricingModelEstimate(family, model)
				if got.PricingUnavailable {
					t.Fatalf("known pricing model resolved unavailable: family=%q model=%q pricing=%#v", family, model, got)
				}
			})
		}
	}
}

func TestBedrockKnownPricingModelsInferBedrockProvider(t *testing.T) {
	cfg := loadPricingConfig()
	if cfg == nil {
		t.Fatal("loadPricingConfig() = nil")
	}
	provider, ok := cfg.provider("bedrock")
	if !ok {
		t.Fatal("pricing config missing bedrock provider")
	}

	for _, model := range provider.KnownModels.Exact {
		t.Run(model, func(t *testing.T) {
			if got := llmcatalog.InferProviderFromModel(model); got != "bedrock" {
				t.Fatalf("InferProviderFromModel(%q) = %q, want bedrock", model, got)
			}
		})
	}
}

func TestDocumentedCatalogClaudeModelsAreKnownPricingModels(t *testing.T) {
	models := []string{
		"claude-sonnet-4-20250514",
		"claude-sonnet-4-5-20250514",
		"claude-sonnet-4-5-20250929",
		"claude-haiku-4-5-20251001",
		"claude-opus-4-20250514",
		"claude-opus-4-5-20251101",
	}

	for _, model := range models {
		t.Run(model, func(t *testing.T) {
			if !pricingFamilyHasKnownModel("claude", model) {
				t.Fatalf("documented/catalog Claude model %q is not in claude known_models.exact", model)
			}

			got := GetPricingInfo("claude", model)
			if got.PricingUnavailable {
				t.Fatalf("documented/catalog Claude model pricing = %#v, want available", got)
			}
		})
	}
}

func knownPricingModelEstimate(family, model string) PricingInfo {
	if family == "kimi" {
		return getKimiPricing(model)
	}
	return resolvePricingByFamily(family, pricingRequest{
		Model:            model,
		PromptTokenCount: 250000,
	})
}
