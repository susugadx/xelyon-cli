package cost

import "testing"

func TestResolveOpenRouterDelegatedProviderPricing(t *testing.T) {
	tests := []struct {
		name  string
		model string
		want  PricingInfo
	}{
		{name: "claude delegated", model: "anthropic/claude-sonnet-4.6", want: getClaudePricing("anthropic/claude-sonnet-4.6", 250000)},
		{name: "openai delegated", model: "openai/gpt-5.4", want: getOpenAIPricing("openai/gpt-5.4", 250000)},
		{name: "openai gpt-5.5 delegated", model: "openai/gpt-5.5", want: getOpenAIPricing("openai/gpt-5.5", 250000)},
		{name: "openai gpt-5.5-pro delegated", model: "openai/gpt-5.5-pro", want: getOpenAIPricing("openai/gpt-5.5-pro", 250000)},
		{name: "gemini delegated", model: "google/gemini-2.5-pro", want: getGeminiPricing("google/gemini-2.5-pro", 250000)},
		{name: "deepseek delegated", model: "deepseek/deepseek-chat", want: getDeepSeekPricing("deepseek/deepseek-chat")},
		{name: "deepseek v4 flash delegated", model: "deepseek/deepseek-v4-flash", want: getDeepSeekPricing("deepseek/deepseek-v4-flash")},
		{name: "deepseek v4 pro delegated", model: "deepseek/deepseek-v4-pro", want: getDeepSeekPricing("deepseek/deepseek-v4-pro")},
		{name: "kimi delegated", model: "moonshotai/kimi-k2.5", want: getKimiPricing("moonshotai/kimi-k2.5")},
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

func TestResolveOpenRouterStaticFallbackPricing(t *testing.T) {
	tests := []struct {
		name      string
		model     string
		wantInput float64
		wantOut   float64
	}{
		{name: "mistral fallback", model: "mistral-medium", wantInput: 2.00, wantOut: 6.00},
		{name: "llama fallback", model: "meta/llama-3.1-70b", wantInput: 0.20, wantOut: 0.80},
		{name: "qwen fallback", model: "qwen/qwen2.5-coder", wantInput: 0.15, wantOut: 0.60},
		{name: "glm fallback", model: "zhipu/glm-5", wantInput: 0.72, wantOut: 2.30},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := resolveOpenRouterStaticFallbackPricing(tt.model)
			if !ok {
				t.Fatalf("resolveOpenRouterStaticFallbackPricing(%q) = not matched", tt.model)
			}
			if got.InputCostPerM != tt.wantInput || got.OutputCostPerM != tt.wantOut {
				t.Fatalf("resolveOpenRouterStaticFallbackPricing(%q) = %#v, want input=%f output=%f", tt.model, got, tt.wantInput, tt.wantOut)
			}
		})
	}
}

func TestResolveOpenRouterStaticFallbackPricing_Unknown(t *testing.T) {
	if _, ok := resolveOpenRouterStaticFallbackPricing("unknown/provider-model"); ok {
		t.Fatal("resolveOpenRouterStaticFallbackPricing(unknown) should not match")
	}
}
