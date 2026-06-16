package cost

import (
	"reflect"
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

	pricing = GetPricingInfo("kimi", "kimi-k2.7-code")
	if pricing.InputCostPerM != 0.95 ||
		pricing.OutputCostPerM != 4.00 ||
		pricing.CachedInputCostPerM != 0.19 ||
		pricing.CacheCreationCostPerM != 0.95 {
		t.Fatalf("Kimi K2.7 hardcoded fallback pricing = %+v, want input=0.95 output=4.00 cached=0.19 create=0.95", pricing)
	}

	pricing = GetPricingInfo("claude", "claude-sonnet-4-5", 250000)
	if pricing.InputCostPerM != 6.00 ||
		pricing.OutputCostPerM != 22.50 ||
		pricing.CachedInputCostPerM != 0.60 ||
		pricing.CacheCreationCostPerM != 7.50 {
		t.Fatalf("Claude Sonnet 4.5 hardcoded fallback long pricing = %+v, want input=6.00 output=22.50 cached=0.60 create=7.50", pricing)
	}

	pricing = GetPricingInfo("claude", "claude-opus-4-5", 250000)
	if pricing.InputCostPerM != 10.00 ||
		pricing.OutputCostPerM != 37.50 ||
		pricing.CachedInputCostPerM != 1.00 ||
		pricing.CacheCreationCostPerM != 12.50 {
		t.Fatalf("Claude Opus 4.5 hardcoded fallback long pricing = %+v, want input=10.00 output=37.50 cached=1.00 create=12.50", pricing)
	}

	pricing = GetPricingInfo("claude", "claude-fable-5", 250000)
	if pricing.InputCostPerM != 10.00 ||
		pricing.OutputCostPerM != 50.00 ||
		pricing.CachedInputCostPerM != 1.00 ||
		pricing.CacheCreationCostPerM != 12.50 {
		t.Fatalf("Claude Fable 5 hardcoded fallback pricing = %+v, want input=10.00 output=50.00 cached=1.00 create=12.50", pricing)
	}

	pricing = GetPricingInfo("claude", "claude-opus-4-8", 250000)
	if pricing.InputCostPerM != 5.00 ||
		pricing.OutputCostPerM != 25.00 ||
		pricing.CachedInputCostPerM != 0.50 ||
		pricing.CacheCreationCostPerM != 6.25 {
		t.Fatalf("Claude Opus 4.8 hardcoded fallback pricing = %+v, want input=5.00 output=25.00 cached=0.50 create=6.25", pricing)
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
		{name: "bedrock invalid sonnet 4.6 v1", family: "bedrock", model: "global.anthropic.claude-sonnet-4-6-v1", want: false},
		{name: "bedrock invalid sonnet 4.6 v1 versioned", family: "bedrock", model: "global.anthropic.claude-sonnet-4-6-v1:0", want: false},
		{name: "bedrock opus 4.8 global exact", family: "bedrock", model: "global.anthropic.claude-opus-4-8", want: true},
		{name: "bedrock opus 4.8 jp exact", family: "bedrock", model: "jp.anthropic.claude-opus-4-8", want: true},
		{name: "bedrock alias contains claude", family: "bedrock", model: "corp-claude-prod", want: false},
		{name: "kimi k2.7 exact", family: "kimi", model: "kimi-k2.7-code", want: true},
		{name: "kimi exact", family: "kimi", model: "kimi-k2.6", want: true},
		{name: "kimi thinking exact", family: "kimi", model: "kimi-k2-thinking", want: true},
		{name: "kimi alias contains exact", family: "kimi", model: "corp-kimi-k2.6-prod", want: false},
		{name: "claude fable exact", family: "claude", model: "claude-fable-5", want: true},
		{name: "openrouter exact", family: "openrouter", model: "openai/gpt-5.4", want: true},
		{name: "openrouter claude opus 4.8 exact", family: "openrouter", model: "anthropic/claude-opus-4.8", want: true},
		{name: "openrouter claude fable unsupported", family: "openrouter", model: "anthropic/claude-fable-5", want: false},
		{name: "openrouter gpt-5.3-codex exact", family: "openrouter", model: "openai/gpt-5.3-codex", want: true},
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

func TestGeminiServiceTierKnownModelsStayInSync(t *testing.T) {
	cfg := loadPricingConfig()
	if cfg == nil {
		t.Fatal("loadPricingConfig() = nil")
	}

	flex, ok := cfg.provider(geminiPricingFamilyFlex)
	if !ok {
		t.Fatalf("pricing config missing %q provider", geminiPricingFamilyFlex)
	}
	priority, ok := cfg.provider(geminiPricingFamilyPriority)
	if !ok {
		t.Fatalf("pricing config missing %q provider", geminiPricingFamilyPriority)
	}
	if !reflect.DeepEqual(flex.KnownModels.Exact, priority.KnownModels.Exact) {
		t.Fatalf("Gemini service tier known models drifted:\nflex=%#v\npriority=%#v", flex.KnownModels.Exact, priority.KnownModels.Exact)
	}

	for _, model := range []string{
		"gemini-2.5-pro-preview",
		"gemini-3-pro-preview",
		"gemini-3.1-pro-preview",
		"gemini-3.1-pro-preview-customtools",
	} {
		if !pricingFamilyHasKnownModel(geminiPricingFamilyFlex, model) {
			t.Fatalf("Gemini flex known models missing %q", model)
		}
		if !pricingFamilyHasKnownModel(geminiPricingFamilyPriority, model) {
			t.Fatalf("Gemini priority known models missing %q", model)
		}
	}
}

func TestGeminiCatalogModelsResolveStandardPricingOrAreExplicitLifecycleOnly(t *testing.T) {
	lifecycleOnlyModels := map[string]string{
		"gemini-1.5-flash": "shutdown model kept for diagnostics; no safe standard pricing rule",
	}

	for _, model := range llmcatalog.KnownModelNamesForProvider("gemini") {
		t.Run(model, func(t *testing.T) {
			got := GetPricingInfo("gemini", model, 250000)
			if got.PricingUnavailable {
				if _, ok := lifecycleOnlyModels[model]; ok {
					return
				}
				t.Fatalf("Gemini catalog model %q is missing standard pricing allowlist coverage", model)
			}
		})
	}

	for model := range lifecycleOnlyModels {
		t.Run("lifecycle-only/"+model, func(t *testing.T) {
			if !llmcatalog.IsExactKnownModelNameForProvider("gemini", model) {
				t.Fatalf("lifecycle-only Gemini model %q is not in the exact catalog", model)
			}
			if got := GetPricingInfo("gemini", model); !got.PricingUnavailable {
				t.Fatalf("lifecycle-only Gemini model %q pricing = %#v, want unavailable", model, got)
			}
		})
	}
}

func TestHasKnownPricingModelUsesProviderPricingFamily(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		model    string
		want     bool
	}{
		{name: "claude dated exact", provider: "claude", model: "claude-sonnet-4-20250514", want: true},
		{name: "claude opus 3 exact", provider: "claude", model: "claude-3-opus-20240229", want: true},
		{name: "claude alias rejected", provider: "claude", model: "corp-claude-sonnet-prod", want: false},
		{name: "claude rejects openai exact", provider: "claude", model: "gpt-5.5", want: false},
		{name: "unknown provider", provider: "unknown", model: "claude-sonnet-4-20250514", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := HasKnownPricingModel(tt.provider, tt.model); got != tt.want {
				t.Fatalf("HasKnownPricingModel(%q, %q) = %v, want %v", tt.provider, tt.model, got, tt.want)
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
		"claude-sonnet-4-6",
		"claude-sonnet-4-5-20250514",
		"claude-sonnet-4-5-20250929",
		"claude-haiku-4-5-20251001",
		"claude-opus-4-20250514",
		"claude-opus-4-8",
		"claude-opus-4-5-20251101",
		"claude-fable-5",
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
