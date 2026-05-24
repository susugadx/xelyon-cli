package cost

import "testing"

func TestResolveOpenRouterDelegatedProviderPricing(t *testing.T) {
	tests := []struct {
		name  string
		model string
		want  PricingInfo
	}{
		{name: "claude delegated", model: "anthropic/claude-sonnet-4.6", want: getClaudePricing("claude-sonnet-4.6", 250000)},
		{name: "openai delegated", model: "openai/gpt-5.4", want: getOpenAIPricing("gpt-5.4", 250000)},
		{name: "openai gpt-5.5 delegated", model: "openai/gpt-5.5", want: getOpenAIPricing("gpt-5.5", 250000)},
		{name: "openai gpt-5.5-pro delegated", model: "openai/gpt-5.5-pro", want: getOpenAIPricing("gpt-5.5-pro", 250000)},
		{name: "gemini delegated", model: "google/gemini-2.5-pro", want: getGeminiStandardPricing("gemini-2.5-pro", 250000)},
		{name: "deepseek delegated", model: "deepseek/deepseek-chat", want: getDeepSeekPricing("deepseek-chat")},
		{name: "deepseek v4 flash delegated", model: "deepseek/deepseek-v4-flash", want: getDeepSeekPricing("deepseek-v4-flash")},
		{name: "deepseek v4 pro delegated", model: "deepseek/deepseek-v4-pro", want: getDeepSeekPricing("deepseek-v4-pro")},
		{name: "kimi delegated", model: "moonshotai/kimi-k2.5", want: getKimiPricing("kimi-k2.5")},
		{name: "kimi k2.6 delegated", model: "moonshotai/kimi-k2.6", want: getKimiPricing("kimi-k2.6")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := resolveOpenRouterDelegatedProviderPricing(tt.model, 250000)
			if !ok {
				t.Fatalf("resolveOpenRouterDelegatedProviderPricing(%q) = not delegated", tt.model)
			}
			if got != tt.want {
				t.Fatalf("resolveOpenRouterDelegatedProviderPricing(%q) = %#v, want %#v", tt.model, got, tt.want)
			}
		})
	}
}

func TestResolveOpenRouterDelegatedProviderPricing_Unknown(t *testing.T) {
	if _, ok := resolveOpenRouterDelegatedProviderPricing("zhipu/glm-5", 0); ok {
		t.Fatal("resolveOpenRouterDelegatedProviderPricing(glm-5) should not delegate")
	}
}

func TestResolveOpenRouterDelegatedProviderPricing_AliasWithKnownSubstringDoesNotDelegate(t *testing.T) {
	tests := []string{
		"openai/gpt-5-prod",
		"openai/gpt-5.3",
		"anthropic/claude-sonnet-prod",
		"google/gemini-pro-prod",
		"moonshotai/kimi-k2-prod",
	}

	for _, model := range tests {
		t.Run(model, func(t *testing.T) {
			if _, ok := resolveOpenRouterDelegatedProviderPricing(model, 0); ok {
				t.Fatalf("resolveOpenRouterDelegatedProviderPricing(%q) should not delegate", model)
			}
		})
	}
}

func TestResolveOpenRouterStaticPricing(t *testing.T) {
	tests := []struct {
		name      string
		model     string
		wantInput float64
		wantOut   float64
	}{
		{name: "mistral static", model: "mistral-medium", wantInput: 2.00, wantOut: 6.00},
		{name: "llama static", model: "meta/llama-3.1-70b", wantInput: 0.20, wantOut: 0.80},
		{name: "qwen static", model: "qwen/qwen2.5-coder", wantInput: 0.15, wantOut: 0.60},
		{name: "glm static", model: "zhipu/glm-5", wantInput: 0.72, wantOut: 2.30},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := resolveOpenRouterStaticPricing(tt.model)
			if !ok {
				t.Fatalf("resolveOpenRouterStaticPricing(%q) = not matched", tt.model)
			}
			if got.InputCostPerM != tt.wantInput || got.OutputCostPerM != tt.wantOut {
				t.Fatalf("resolveOpenRouterStaticPricing(%q) = %#v, want input=%f output=%f", tt.model, got, tt.wantInput, tt.wantOut)
			}
		})
	}
}

func TestOpenRouterStaticPricingRulesAreAllowlisted(t *testing.T) {
	for _, rule := range openRouterStaticPricingRules {
		for _, model := range rule.modelIDs {
			t.Run(model, func(t *testing.T) {
				if !pricingFamilyHasKnownModel("openrouter", model) {
					t.Fatalf("OpenRouter static pricing model %q is not in openrouter known_models.exact", model)
				}
			})
		}
	}
}

func TestResolveOpenRouterStaticPricing_Unknown(t *testing.T) {
	if _, ok := resolveOpenRouterStaticPricing("unknown/provider-model"); ok {
		t.Fatal("resolveOpenRouterStaticPricing(unknown) should not match")
	}
}

func TestResolveOpenRouterStaticPricing_AliasWithKnownSubstringDoesNotMatch(t *testing.T) {
	tests := []string{
		"meta/llama-prod",
		"qwen/qwen-prod",
		"zhipu/glm-5-prod",
		"corp/mistral-prod",
	}

	for _, model := range tests {
		t.Run(model, func(t *testing.T) {
			if _, ok := resolveOpenRouterStaticPricing(model); ok {
				t.Fatalf("resolveOpenRouterStaticPricing(%q) should not match", model)
			}
		})
	}
}

func TestGetOpenRouterPricing_UnknownUnavailable(t *testing.T) {
	got := getOpenRouterPricing("unknown/provider-model", 0)
	if !got.PricingUnavailable {
		t.Fatalf("getOpenRouterPricing(unknown).PricingUnavailable = false, want true: %#v", got)
	}
}

func TestGetOpenRouterPricing_NonAllowlistedDelegatedIDUnavailable(t *testing.T) {
	got := getOpenRouterPricing("openai/gpt-5.3", 0)
	if !got.PricingUnavailable {
		t.Fatalf("getOpenRouterPricing(non-allowlisted delegated ID).PricingUnavailable = false, want true: %#v", got)
	}
}
