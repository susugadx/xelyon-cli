package cost

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

// --- GetOpenAIPricing table-driven tests ---

func TestGetOpenAIPricing_AllModels(t *testing.T) {
	tests := []struct {
		name       string
		model      string
		ptc        int
		wantInput  float64
		wantOutput float64
		wantCached float64
	}{
		{name: "gpt-5.4-nano", model: "gpt-5.4-nano", wantInput: 0.20, wantOutput: 1.25, wantCached: 0.02},
		{name: "gpt-5.4", model: "gpt-5.4", wantInput: 2.50, wantOutput: 15.00, wantCached: 0.25},
		{name: "gpt-5.4-pro", model: "gpt-5.4-pro", wantInput: 30.00, wantOutput: 180.00, wantCached: 3.00},
		{name: "gpt-5.2-codex", model: "gpt-5.2-codex", wantInput: 1.75, wantOutput: 14.00, wantCached: 0.175},
		{name: "gpt-5-mini", model: "gpt-5-mini", wantInput: 0.25, wantOutput: 2.00, wantCached: 0.025},
		{name: "gpt-5.4-mini", model: "gpt-5.4-mini", wantInput: 0.75, wantOutput: 4.50, wantCached: 0.075},
		{name: "gpt-5.1", model: "gpt-5.1", wantInput: 2.00, wantOutput: 8.00, wantCached: 0.50},
		{name: "gpt-5.3", model: "gpt-5.3", wantInput: 1.75, wantOutput: 14.00, wantCached: 0.175},
		{name: "gpt-5.2-pro", model: "gpt-5.2-pro", wantInput: 21.00, wantOutput: 168.00, wantCached: 2.10},
		{name: "gpt-5 default", model: "gpt-5", wantInput: 1.25, wantOutput: 10.00, wantCached: 0.125},
		{name: "generic nano", model: "gpt-5-nano", wantInput: 0.05, wantOutput: 0.40, wantCached: 0.005},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pricing := getOpenAIPricing(tt.model, tt.ptc)
			if pricing.InputCostPerM != tt.wantInput {
				t.Errorf("InputCostPerM = %f, want %f", pricing.InputCostPerM, tt.wantInput)
			}
			if pricing.OutputCostPerM != tt.wantOutput {
				t.Errorf("OutputCostPerM = %f, want %f", pricing.OutputCostPerM, tt.wantOutput)
			}
			if pricing.CachedInputCostPerM != tt.wantCached {
				t.Errorf("CachedInputCostPerM = %f, want %f", pricing.CachedInputCostPerM, tt.wantCached)
			}
		})
	}
}

// --- GetClaudePricing table-driven tests ---

func TestGetClaudePricing_AllModels(t *testing.T) {
	tests := []struct {
		name       string
		model      string
		ptc        int
		wantInput  float64
		wantOutput float64
		wantCached float64
	}{
		{name: "opus normal", model: "claude-opus-4-6", ptc: 0, wantInput: 5.00, wantOutput: 25.00, wantCached: 0.50},
		{name: "sonnet normal", model: "claude-sonnet-4-5", ptc: 0, wantInput: 3.00, wantOutput: 15.00, wantCached: 0.30},
		{name: "haiku normal", model: "claude-haiku-4-5", ptc: 0, wantInput: 1.00, wantOutput: 5.00, wantCached: 0.10},
		{name: "opus long context", model: "claude-opus-4-6", ptc: 250000, wantInput: 10.00, wantOutput: 37.50, wantCached: 1.00},
		{name: "sonnet long context", model: "claude-sonnet-4-6", ptc: 250000, wantInput: 6.00, wantOutput: 22.50, wantCached: 0.60},
		{name: "haiku ignores long context", model: "claude-haiku-4-5", ptc: 250000, wantInput: 1.00, wantOutput: 5.00, wantCached: 0.10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pricing := getClaudePricing(tt.model, tt.ptc)
			if pricing.InputCostPerM != tt.wantInput {
				t.Errorf("InputCostPerM = %f, want %f", pricing.InputCostPerM, tt.wantInput)
			}
			if pricing.OutputCostPerM != tt.wantOutput {
				t.Errorf("OutputCostPerM = %f, want %f", pricing.OutputCostPerM, tt.wantOutput)
			}
			if pricing.CachedInputCostPerM != tt.wantCached {
				t.Errorf("CachedInputCostPerM = %f, want %f", pricing.CachedInputCostPerM, tt.wantCached)
			}
		})
	}
}

// --- GetGeminiPricing table-driven tests ---

func TestGetGeminiPricing_AllModels(t *testing.T) {
	tests := []struct {
		name       string
		model      string
		ptc        int
		wantInput  float64
		wantOutput float64
	}{
		{name: "2.5-pro normal", model: "gemini-2.5-pro", ptc: 100000, wantInput: 1.25, wantOutput: 10.00},
		{name: "2.5-pro long", model: "gemini-2.5-pro", ptc: 250000, wantInput: 2.50, wantOutput: 15.00},
		{name: "2.5-flash", model: "gemini-2.5-flash", ptc: 0, wantInput: 0.30, wantOutput: 2.50},
		{name: "3.x flash default", model: "gemini-3-flash", ptc: 0, wantInput: 0.50, wantOutput: 3.00},
		{name: "3.1-pro normal", model: "gemini-3.1-pro", ptc: 100000, wantInput: 2.00, wantOutput: 12.00},
		{name: "3.1-pro long", model: "gemini-3.1-pro", ptc: 250000, wantInput: 4.00, wantOutput: 18.00},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pricing := getGeminiPricing(tt.model, tt.ptc)
			if pricing.InputCostPerM != tt.wantInput {
				t.Errorf("InputCostPerM = %f, want %f", pricing.InputCostPerM, tt.wantInput)
			}
			if pricing.OutputCostPerM != tt.wantOutput {
				t.Errorf("OutputCostPerM = %f, want %f", pricing.OutputCostPerM, tt.wantOutput)
			}
		})
	}
}

// --- GetGroqPricing table-driven tests ---

func TestGetGroqPricing_AllModels(t *testing.T) {
	tests := []struct {
		name       string
		model      string
		wantInput  float64
		wantOutput float64
	}{
		{name: "llama-3-70b", model: "llama-3-70b", wantInput: 0.59, wantOutput: 0.79},
		{name: "llama-3.1-70b", model: "llama-3.1-70b", wantInput: 0.59, wantOutput: 0.79},
		{name: "mixtral-8x7b", model: "mixtral-8x7b", wantInput: 0.24, wantOutput: 0.24},
		{name: "llama-3.1-405b", model: "llama-3.1-405b", wantInput: 2.00, wantOutput: 2.00},
		{name: "gemma-7b", model: "gemma-7b", wantInput: 0.07, wantOutput: 0.07},
		{name: "default (llama 8b)", model: "llama-3-8b", wantInput: 0.05, wantOutput: 0.10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pricing := getGroqPricing(tt.model)
			if pricing.InputCostPerM != tt.wantInput {
				t.Errorf("InputCostPerM = %f, want %f", pricing.InputCostPerM, tt.wantInput)
			}
			if pricing.OutputCostPerM != tt.wantOutput {
				t.Errorf("OutputCostPerM = %f, want %f", pricing.OutputCostPerM, tt.wantOutput)
			}
		})
	}
}

// --- GetPricingInfo routing tests ---

func TestGetPricingInfo_ProviderRouting(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		model    string
	}{
		{name: "deepseek routes correctly", provider: "deepseek", model: "deepseek-chat"},
		{name: "claude routes correctly", provider: "claude", model: "claude-sonnet-4-5"},
		{name: "anthropic alias routes like claude", provider: "anthropic", model: "claude-sonnet-4-5"},
		{name: "openai routes correctly", provider: "openai", model: "gpt-5.4"},
		{name: "gemini routes correctly", provider: "gemini", model: "gemini-2.5-pro"},
		{name: "groq routes correctly", provider: "groq", model: "llama-3-70b"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pricing := GetPricingInfo(tt.provider, tt.model)
			if pricing.InputCostPerM <= 0 {
				t.Errorf("GetPricingInfo(%q, %q).InputCostPerM = %f, want > 0", tt.provider, tt.model, pricing.InputCostPerM)
			}
			if pricing.OutputCostPerM <= 0 {
				t.Errorf("GetPricingInfo(%q, %q).OutputCostPerM = %f, want > 0", tt.provider, tt.model, pricing.OutputCostPerM)
			}
		})
	}
}

func TestGetPricingInfo_OllamaIsZero(t *testing.T) {
	pricing := GetPricingInfo("ollama", "llama3")
	if pricing.InputCostPerM != 0 {
		t.Errorf("ollama InputCostPerM = %f, want 0", pricing.InputCostPerM)
	}
	if pricing.OutputCostPerM != 0 {
		t.Errorf("ollama OutputCostPerM = %f, want 0", pricing.OutputCostPerM)
	}
}

func TestGetPricingInfo_UnknownProviderFallback(t *testing.T) {
	pricing := GetPricingInfo("unknown-provider", "some-model")
	// Should return DeepSeek V3.2 fallback pricing
	if pricing.InputCostPerM != 0.28 {
		t.Errorf("unknown provider InputCostPerM = %f, want 0.28", pricing.InputCostPerM)
	}
	if pricing.OutputCostPerM != 0.42 {
		t.Errorf("unknown provider OutputCostPerM = %f, want 0.42", pricing.OutputCostPerM)
	}
}

func TestGetBedrockPricing_ClaudeDelegation(t *testing.T) {
	model := "global.anthropic.claude-sonnet-4-6-v1:0"
	promptTokens := 250000

	got := getBedrockPricing(model, promptTokens)
	want := getClaudePricing(model, promptTokens)

	if got != want {
		t.Fatalf("getBedrockPricing() = %#v, want %#v", got, want)
	}
}

func TestGetBedrockPricing_NonClaudeFallsBack(t *testing.T) {
	got := getBedrockPricing("amazon.nova-pro-v1:0", 0)
	want := getUnknownProviderFallbackPricing()

	if got != want {
		t.Fatalf("getBedrockPricing(non-claude) = %#v, want %#v", got, want)
	}
}

func TestGetPricingInfo_OpenRouter_GLM5(t *testing.T) {
	pricing := GetPricingInfo("openrouter", "zhipu/glm-5")
	if pricing.InputCostPerM != 0.72 {
		t.Errorf("openrouter glm-5 InputCostPerM = %f, want 0.72", pricing.InputCostPerM)
	}
	if pricing.OutputCostPerM != 2.30 {
		t.Errorf("openrouter glm-5 OutputCostPerM = %f, want 2.30", pricing.OutputCostPerM)
	}
	if pricing.CachedInputCostPerM != 0.072 {
		t.Errorf("openrouter glm-5 CachedInputCostPerM = %f, want 0.072", pricing.CachedInputCostPerM)
	}
}

func TestGetPricingInfoForConfig_UsesCatalogModel(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("openai", config.ProviderModelConfig{
		DefaultModel: "corp-gpt-deployment",
		CatalogModel: "gpt-5.4-mini",
		ModelOverrides: map[string]config.ModelOverride{
			"pro-deployment": {CatalogModel: "gpt-5.4-pro"},
		},
	})

	defaultDeployment := GetPricingInfoForConfig(cfg, "openai", "corp-gpt-deployment")
	if defaultDeployment.InputCostPerM != 0.75 || defaultDeployment.OutputCostPerM != 4.50 {
		t.Fatalf("default deployment pricing = %#v, want gpt-5.4-mini pricing", defaultDeployment)
	}

	overrideDeployment := GetPricingInfoForConfig(cfg, "openai", "pro-deployment")
	if overrideDeployment.InputCostPerM != 30.00 || overrideDeployment.OutputCostPerM != 180.00 {
		t.Fatalf("override deployment pricing = %#v, want gpt-5.4-pro pricing", overrideDeployment)
	}
}

func TestResolveProviderPricingFromConfig_UsesLongInputTierWhenEnabled(t *testing.T) {
	provider := providerPricingConfig{
		Default: PricingInfo{InputCostPerM: 1.0},
		LongInput: &longInputTier{
			Threshold: 200,
			Pricing:   PricingInfo{InputCostPerM: 2.0},
		},
	}

	got := resolveProviderPricingFromConfig(provider, "model", 201, true)
	if got.InputCostPerM != 2.0 {
		t.Fatalf("resolveProviderPricingFromConfig(long enabled) = %#v, want long-input tier", got)
	}
}

func TestResolveProviderPricingFromConfig_DoesNotUseLongInputTierWhenDisabled(t *testing.T) {
	provider := providerPricingConfig{
		Default: PricingInfo{InputCostPerM: 1.0},
		LongInput: &longInputTier{
			Threshold: 200,
			Pricing:   PricingInfo{InputCostPerM: 2.0},
		},
	}

	got := resolveProviderPricingFromConfig(provider, "model", 201, false)
	if got.InputCostPerM != 1.0 {
		t.Fatalf("resolveProviderPricingFromConfig(long disabled) = %#v, want default tier", got)
	}
}

// --- CalculateRequestCost tests ---

func TestCalculateRequestCost_BasicCost(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		model    string
		input    int
		output   int
		wantMin  float64
		wantMax  float64
	}{
		{
			name: "deepseek 1M tokens", provider: "deepseek", model: "deepseek-chat",
			input: 1000000, output: 1000000, wantMin: 0.69, wantMax: 0.71,
		},
		{
			name: "ollama always zero", provider: "ollama", model: "llama3",
			input: 1000000, output: 1000000, wantMin: 0, wantMax: 0,
		},
		{
			name: "zero tokens", provider: "deepseek", model: "deepseek-chat",
			input: 0, output: 0, wantMin: 0, wantMax: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cost := CalculateRequestCost(tt.provider, tt.model, tt.input, tt.output)
			if cost < tt.wantMin || cost > tt.wantMax {
				t.Errorf("CalculateRequestCost() = %f, want between %f and %f", cost, tt.wantMin, tt.wantMax)
			}
		})
	}
}

// --- CalculateRequestCostWithCache tests ---

func TestCalculateRequestCostWithCache_Basic(t *testing.T) {
	// Claude Sonnet: 100K input (50K cached, 25K creation, 25K uncached), 50K output
	cost := CalculateRequestCostWithCache("claude", "claude-sonnet-4-5", api.Usage{
		InputTokens:         100000,
		OutputTokens:        50000,
		CachedInputTokens:   50000,
		CacheCreationTokens: 25000,
	})
	// totalInputForTier = 100K + 50K + 25K = 175K <= 200K -> normal pricing
	// uncached: (100K - 50K - 25K) = 25K / 1M * $3.00 = $0.075
	// cached: 50K / 1M * $0.30 = $0.015
	// creation: 25K / 1M * $3.75 = $0.09375
	// output: 50K / 1M * $15.00 = $0.75
	// total: ~$0.934
	if cost < 0.92 || cost > 0.94 {
		t.Errorf("CalculateRequestCostWithCache() = %f, want ~0.934", cost)
	}
}

func TestCalculateRequestCostWithCache_OllamaZero(t *testing.T) {
	cost := CalculateRequestCostWithCache("ollama", "llama3", api.Usage{
		InputTokens:  100000,
		OutputTokens: 50000,
	})
	if cost != 0.0 {
		t.Errorf("ollama cost = %f, want 0.0", cost)
	}
}
