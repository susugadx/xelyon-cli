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
		{name: "gpt-5.5", model: "gpt-5.5", ptc: 100000, wantInput: 5.00, wantOutput: 30.00, wantCached: 0.50},
		{name: "gpt-5.5 long input", model: "gpt-5.5", ptc: 300000, wantInput: 10.00, wantOutput: 45.00, wantCached: 1.00},
		{name: "gpt-5.5-pro", model: "gpt-5.5-pro", ptc: 100000, wantInput: 30.00, wantOutput: 180.00, wantCached: 30.00},
		{name: "gpt-5.5-pro long input keeps standard pricing", model: "gpt-5.5-pro", ptc: 300000, wantInput: 30.00, wantOutput: 180.00, wantCached: 30.00},
		{name: "gpt-5.4-nano", model: "gpt-5.4-nano", wantInput: 0.20, wantOutput: 1.25, wantCached: 0.02},
		{name: "gpt-5.4", model: "gpt-5.4", wantInput: 2.50, wantOutput: 15.00, wantCached: 0.25},
		{name: "gpt-5.4-pro", model: "gpt-5.4-pro", wantInput: 30.00, wantOutput: 180.00, wantCached: 3.00},
		{name: "gpt-5.2-codex", model: "gpt-5.2-codex", wantInput: 1.75, wantOutput: 14.00, wantCached: 0.175},
		{name: "codex-mini", model: "codex-mini", wantInput: 0.25, wantOutput: 2.00, wantCached: 0.025},
		{name: "gpt-5.4-mini", model: "gpt-5.4-mini", wantInput: 0.75, wantOutput: 4.50, wantCached: 0.075},
		{name: "gpt-5.1", model: "gpt-5.1", wantInput: 2.00, wantOutput: 8.00, wantCached: 0.50},
		{name: "gpt-5.3-codex", model: "gpt-5.3-codex", wantInput: 1.75, wantOutput: 14.00, wantCached: 0.175},
		{name: "gpt-5.2-pro", model: "gpt-5.2-pro", wantInput: 21.00, wantOutput: 168.00, wantCached: 2.10},
		{name: "gpt-5 default", model: "gpt-5", wantInput: 1.25, wantOutput: 10.00, wantCached: 0.125},
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

func TestGetOpenAIPricing_RuleMatchWithoutKnownExactUnavailable(t *testing.T) {
	tests := []string{
		"gpt-5.3",
		"gpt-5-nano",
		"corp-gpt-5.3-codex-prod",
	}

	for _, model := range tests {
		t.Run(model, func(t *testing.T) {
			pricing := getOpenAIPricing(model, 0)
			if !pricing.PricingUnavailable {
				t.Fatalf("getOpenAIPricing(%q).PricingUnavailable = false, want true: %#v", model, pricing)
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
			if pricing.PricingUnavailable {
				t.Fatalf("GetPricingInfo(%q, %q).PricingUnavailable = true, want false", tt.provider, tt.model)
			}
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
	if pricing.PricingUnavailable {
		t.Error("ollama PricingUnavailable = true, want false")
	}
}

func TestGetPricingInfo_UnknownProviderUnavailable(t *testing.T) {
	pricing := GetPricingInfo("unknown-provider", "some-model")
	if !pricing.PricingUnavailable {
		t.Fatalf("unknown provider PricingUnavailable = false, want true: %#v", pricing)
	}
	if pricing.InputCostPerM != 0 || pricing.OutputCostPerM != 0 {
		t.Fatalf("unknown provider pricing = %#v, want zero rates with PricingUnavailable", pricing)
	}
}

func TestGetDeepSeekPricing_V4ModelsAndLegacyAliases(t *testing.T) {
	tests := []struct {
		name       string
		model      string
		wantInput  float64
		wantOutput float64
		wantCached float64
	}{
		{name: "v4 flash", model: "deepseek-v4-flash", wantInput: 0.14, wantOutput: 0.28, wantCached: 0.0028},
		{name: "v4 pro", model: "deepseek-v4-pro", wantInput: 1.74, wantOutput: 3.48, wantCached: 0.0145},
		{name: "legacy chat", model: "deepseek-chat", wantInput: 0.14, wantOutput: 0.28, wantCached: 0.0028},
		{name: "legacy reasoner", model: "deepseek-reasoner", wantInput: 0.14, wantOutput: 0.28, wantCached: 0.0028},
		{name: "v3 static", model: "deepseek-v3", wantInput: 0.28, wantOutput: 0.42, wantCached: 0.028},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pricing := getDeepSeekPricing(tt.model)
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

func TestGetBedrockPricing_ClaudeCatalogAliasDelegation(t *testing.T) {
	model := "claude-sonnet-4-6"
	promptTokens := 250000

	got := getBedrockPricing(model, promptTokens)
	if got.PricingUnavailable {
		t.Fatalf("getBedrockPricing() = %#v, want available Claude pricing", got)
	}
	if got.InputCostPerM != 6.00 || got.OutputCostPerM != 22.50 {
		t.Fatalf("getBedrockPricing() = %#v, want long-context Sonnet pricing", got)
	}
}

func TestGetBedrockPricing_BedrockClaudeModelsFromAWSPriceList(t *testing.T) {
	tests := []struct {
		name      string
		model     string
		prompt    int
		wantInput float64
		wantOut   float64
	}{
		{name: "global sonnet 4.6", model: "global.anthropic.claude-sonnet-4-6", wantInput: 3.00, wantOut: 15.00},
		{name: "geo sonnet 4.6", model: "us.anthropic.claude-sonnet-4-6", wantInput: 3.30, wantOut: 16.50},
		{name: "direct sonnet 4.6", model: "anthropic.claude-sonnet-4-6", wantInput: 3.30, wantOut: 16.50},
		{name: "global sonnet 4.5 long context", model: "global.anthropic.claude-sonnet-4-5-20250929-v1:0", prompt: 250000, wantInput: 6.00, wantOut: 22.50},
		{name: "direct sonnet 4.5 long context", model: "anthropic.claude-sonnet-4-5-20250929-v1:0", prompt: 250000, wantInput: 6.60, wantOut: 24.75},
		{name: "direct sonnet 4", model: "anthropic.claude-sonnet-4-20250514-v1:0", wantInput: 3.00, wantOut: 15.00},
		{name: "global sonnet 4 profile", model: "global.anthropic.claude-sonnet-4-20250514-v1:0", wantInput: 3.00, wantOut: 15.00},
		{name: "direct sonnet 3 keeps standard pricing", model: "anthropic.claude-3-sonnet-20240229-v1:0:200k", prompt: 250000, wantInput: 3.00, wantOut: 15.00},
		{name: "direct haiku 4.5", model: "anthropic.claude-haiku-4-5-20251001-v1:0", wantInput: 1.10, wantOut: 5.50},
		{name: "global haiku 4.5 profile", model: "global.anthropic.claude-haiku-4-5-20251001-v1:0", wantInput: 1.00, wantOut: 5.00},
		{name: "direct haiku 3.5", model: "anthropic.claude-3-5-haiku-20241022-v1:0", wantInput: 0.80, wantOut: 4.00},
		{name: "us sonnet 3.7 profile", model: "us.anthropic.claude-3-7-sonnet-20250219-v1:0", wantInput: 3.00, wantOut: 15.00},
		{name: "direct haiku 3", model: "anthropic.claude-3-haiku-20240307-v1:0:200k", wantInput: 0.25, wantOut: 1.25},
		{name: "us opus 3 profile", model: "us.anthropic.claude-3-opus-20240229-v1:0", wantInput: 15.00, wantOut: 75.00},
		{name: "direct opus 4", model: "anthropic.claude-opus-4-20250514-v1:0", wantInput: 15.00, wantOut: 75.00},
		{name: "direct opus 4.5", model: "anthropic.claude-opus-4-5-20251101-v1:0", wantInput: 5.50, wantOut: 27.50},
		{name: "global opus 4.6 profile", model: "global.anthropic.claude-opus-4-6-v1", wantInput: 5.00, wantOut: 25.00},
		{name: "us opus 4.6 profile", model: "us.anthropic.claude-opus-4-6-v1", wantInput: 5.50, wantOut: 27.50},
		{name: "global opus 4.7", model: "global.anthropic.claude-opus-4-7-v1:0", wantInput: 5.00, wantOut: 25.00},
		{name: "global opus 4.7 profile", model: "global.anthropic.claude-opus-4-7", wantInput: 5.00, wantOut: 25.00},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getBedrockPricing(tt.model, tt.prompt)
			if got.PricingUnavailable {
				t.Fatalf("getBedrockPricing(%q) = %#v, want available", tt.model, got)
			}
			if got.InputCostPerM != tt.wantInput || got.OutputCostPerM != tt.wantOut {
				t.Fatalf("getBedrockPricing(%q) = %#v, want input=%f output=%f", tt.model, got, tt.wantInput, tt.wantOut)
			}
		})
	}
}

func TestGetBedrockPricing_ConverseModelsFromAWSPriceList(t *testing.T) {
	tests := []struct {
		name      string
		model     string
		wantInput float64
		wantOut   float64
	}{
		{name: "nova pro", model: "amazon.nova-pro-v1:0", wantInput: 0.80, wantOut: 3.20},
		{name: "nova 2 lite", model: "amazon.nova-2-lite-v1:0", wantInput: 0.33, wantOut: 2.75},
		{name: "global nova 2 lite profile", model: "global.amazon.nova-2-lite-v1:0", wantInput: 0.30, wantOut: 2.50},
		{name: "us nova pro profile", model: "us.amazon.nova-pro-v1:0", wantInput: 0.80, wantOut: 3.20},
		{name: "llama 4 scout", model: "meta.llama4-scout-17b-instruct-v1:0", wantInput: 0.17, wantOut: 0.66},
		{name: "us llama 4 scout profile", model: "us.meta.llama4-scout-17b-instruct-v1:0", wantInput: 0.17, wantOut: 0.66},
		{name: "mistral large 3", model: "mistral.mistral-large-3-675b-instruct", wantInput: 0.50, wantOut: 1.50},
		{name: "us pixtral profile", model: "us.mistral.pixtral-large-2502-v1:0", wantInput: 2.00, wantOut: 6.00},
		{name: "cohere command r plus", model: "cohere.command-r-plus-v1:0", wantInput: 3.00, wantOut: 15.00},
		{name: "ai21 jamba mini", model: "ai21.jamba-1-5-mini-v1:0", wantInput: 0.20, wantOut: 0.40},
		{name: "writer palmyra x5", model: "writer.palmyra-x5-v1:0", wantInput: 0.60, wantOut: 6.00},
		{name: "us writer palmyra x5 profile", model: "us.writer.palmyra-x5-v1:0", wantInput: 0.60, wantOut: 6.00},
		{name: "writer palmyra x4", model: "writer.palmyra-x4-v1:0", wantInput: 2.50, wantOut: 10.00},
		{name: "us deepseek r1 profile", model: "us.deepseek.r1-v1:0", wantInput: 1.35, wantOut: 5.40},
		{name: "deepseek v3.2", model: "deepseek.v3.2", wantInput: 0.62, wantOut: 1.85},
		{name: "qwen next", model: "qwen.qwen3-next-80b-a3b", wantInput: 0.14, wantOut: 1.20},
		{name: "minimax m2", model: "minimax.minimax-m2", wantInput: 0.30, wantOut: 1.20},
		{name: "nvidia nemotron", model: "nvidia.nemotron-nano-3-30b", wantInput: 0.06, wantOut: 0.24},
		{name: "openai gpt oss", model: "openai.gpt-oss-20b-1:0", wantInput: 0.07, wantOut: 0.30},
		{name: "google gemma", model: "google.gemma-3-4b-it", wantInput: 0.04, wantOut: 0.08},
		{name: "moonshot kimi", model: "moonshotai.kimi-k2.5", wantInput: 0.60, wantOut: 3.00},
		{name: "moonshot kimi thinking", model: "moonshotai.kimi-k2-thinking", wantInput: 0.60, wantOut: 2.50},
		{name: "zai glm flash", model: "zai.glm-4.7-flash", wantInput: 0.07, wantOut: 0.40},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getBedrockPricing(tt.model, 0)
			if got.PricingUnavailable {
				t.Fatalf("getBedrockPricing(%q) = %#v, want available", tt.model, got)
			}
			if got.InputCostPerM != tt.wantInput || got.OutputCostPerM != tt.wantOut {
				t.Fatalf("getBedrockPricing(%q) = %#v, want input=%f output=%f", tt.model, got, tt.wantInput, tt.wantOut)
			}
		})
	}
}

func TestGetBedrockPricing_NovaPromptCachePrices(t *testing.T) {
	got := getBedrockPricing("amazon.nova-pro-v1:0", 0)
	if got.PricingUnavailable {
		t.Fatalf("getBedrockPricing(nova pro) = %#v, want available", got)
	}
	if got.CachedInputCostPerM != 0.20 {
		t.Fatalf("Nova Pro cached input = %f, want 0.20", got.CachedInputCostPerM)
	}
	if got.CacheCreationCostPerM != 0 {
		t.Fatalf("Nova Pro cache creation = %f, want 0", got.CacheCreationCostPerM)
	}
}

func TestGetBedrockPricing_UnknownConverseModelUnavailable(t *testing.T) {
	for _, model := range []string{
		"",
		" \t ",
		"amazon.nova-pro-v9:0",
		"global.cohere.embed-v4:0",
		"global.twelvelabs.pegasus-1-2-v1:0",
		"us.stability.stable-image-inpaint-v1:0",
		"us.twelvelabs.marengo-embed-3-0-v1:0",
		"corp-amazon.nova-pro-v1",
		"moonshot.kimi-k2-thinking",
		"qwen.qwen3-next-prod",
	} {
		t.Run(model, func(t *testing.T) {
			got := getBedrockPricing(model, 0)
			if !got.PricingUnavailable {
				t.Fatalf("getBedrockPricing(%q) = %#v, want unavailable", model, got)
			}
		})
	}
}

func TestEstimateRequestCostWithCache_EmptyBedrockModelUnavailable(t *testing.T) {
	estimate := EstimateRequestCostWithCache("bedrock", "", api.Usage{
		InputTokens:  1000000,
		OutputTokens: 1000000,
	})
	if !estimate.PricingUnavailable {
		t.Fatalf("PricingUnavailable = false, want true: %#v", estimate)
	}
	if estimate.Cost != 0 {
		t.Fatalf("Cost = %f, want 0 for unavailable pricing", estimate.Cost)
	}
}

func TestGetPricingInfoForConfig_ConfiguredBedrockClaudeWithoutCatalogStillPrices(t *testing.T) {
	model := "global.anthropic.claude-sonnet-4-6"
	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("bedrock", config.ProviderModelConfig{
		DefaultModel: model,
	})

	got := GetPricingInfoForConfig(cfg, "bedrock", model)
	if got.PricingUnavailable {
		t.Fatalf("configured Bedrock Claude pricing = %#v, want available", got)
	}
	if got.InputCostPerM != 3.00 || got.OutputCostPerM != 15.00 {
		t.Fatalf("configured Bedrock Claude pricing = %#v, want Sonnet pricing", got)
	}
}

func TestGetPricingInfoForConfig_BedrockAliasWithClaudeCatalogModelPrices(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("bedrock", config.ProviderModelConfig{
		DefaultModel: "corp-bedrock-sonnet",
		CatalogModel: "claude-sonnet-4-6",
	})

	got := GetPricingInfoForConfig(cfg, "bedrock", "corp-bedrock-sonnet")
	if got.PricingUnavailable {
		t.Fatalf("Bedrock alias with Claude catalog_model pricing = %#v, want available", got)
	}
	if got.InputCostPerM != 3.00 || got.OutputCostPerM != 15.00 {
		t.Fatalf("Bedrock alias with Claude catalog_model pricing = %#v, want Sonnet pricing", got)
	}
}

func TestGetPricingInfoForConfig_ClaudeAliasWithDatedCatalogModelPrices(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("claude", config.ProviderModelConfig{
		DefaultModel: "corp-sonnet",
		CatalogModel: "claude-sonnet-4-5-20250514",
	})

	got := GetPricingInfoForConfig(cfg, "claude", "corp-sonnet")
	if got.PricingUnavailable {
		t.Fatalf("Claude alias with dated catalog_model pricing = %#v, want available", got)
	}
	if got.InputCostPerM != 3.00 || got.OutputCostPerM != 15.00 {
		t.Fatalf("Claude alias with dated catalog_model pricing = %#v, want Sonnet pricing", got)
	}
}

func TestGetBedrockPricing_ClaudeAliasUnavailable(t *testing.T) {
	got := getBedrockPricing("corp-claude-prod", 0)
	if !got.PricingUnavailable {
		t.Fatalf("getBedrockPricing(alias).PricingUnavailable = false, want true: %#v", got)
	}
}

func TestGetBedrockPricing_UnknownNonClaudeUnavailable(t *testing.T) {
	got := getBedrockPricing("amazon.nova-unknown-v1:0", 0)
	if !got.PricingUnavailable {
		t.Fatalf("getBedrockPricing(non-claude).PricingUnavailable = false, want true: %#v", got)
	}
}

func TestGetPricingInfo_OpenRouter_GLM5(t *testing.T) {
	pricing := GetPricingInfo("openrouter", "zhipu/glm-5")
	if pricing.PricingUnavailable {
		t.Fatalf("openrouter glm-5 PricingUnavailable = true, want false")
	}
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

func TestGetPricingInfoForConfig_DefaultOverrideInheritsProviderCatalogModel(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("openai", config.ProviderModelConfig{
		DefaultModel: "corp-gpt-deployment",
		CatalogModel: "gpt-5.4",
		ModelOverrides: map[string]config.ModelOverride{
			"corp-gpt-deployment": {MaxOutputTokens: 8192},
		},
	})

	got := GetPricingInfoForConfig(cfg, "openai", "corp-gpt-deployment")
	if got.PricingUnavailable {
		t.Fatalf("default override inherited catalog pricing = %#v, want available", got)
	}
	if got.InputCostPerM != 2.50 || got.OutputCostPerM != 15.00 {
		t.Fatalf("default override inherited catalog pricing = %#v, want GPT-5.4 pricing", got)
	}
}

func TestGetPricingInfoForConfig_CustomModelWithoutCatalogIsUnavailable(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("openai", config.ProviderModelConfig{
		DefaultModel: "corp-gpt-deployment",
	})

	got := GetPricingInfoForConfig(cfg, "openai", "corp-gpt-deployment")
	if !got.PricingUnavailable {
		t.Fatalf("custom deployment without catalog pricing = %#v, want unavailable", got)
	}
}

func TestGetPricingInfoForConfig_ConfiguredAliasWithKnownSubstringIsUnavailable(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		model    string
	}{
		{name: "openai deployment contains gpt-5", provider: "openai", model: "corp-gpt-5-prod"},
		{name: "claude alias contains sonnet", provider: "claude", model: "claude-sonnet-prod"},
		{name: "deepseek alias contains v4 pro", provider: "deepseek", model: "corp-deepseek-v4-pro"},
		{name: "gemini alias contains pro", provider: "gemini", model: "gemini-3.1-pro-prod"},
		{name: "groq alias contains 70b", provider: "groq", model: "corp-llama-3.1-70b-prod"},
		{name: "bedrock alias contains claude", provider: "bedrock", model: "corp-claude-sonnet-prod"},
		{name: "bedrock alias contains nova", provider: "bedrock", model: "corp-amazon.nova-pro-v1"},
		{name: "openrouter alias contains delegated openai model", provider: "openrouter", model: "openai/gpt-5-prod"},
		{name: "openrouter alias contains delegated kimi model", provider: "openrouter", model: "moonshotai/kimi-k2-prod"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.SetProviderModelConfig(tt.provider, config.ProviderModelConfig{
				DefaultModel: tt.model,
				ModelOverrides: map[string]config.ModelOverride{
					tt.model + "-override": {MaxOutputTokens: 2048},
				},
			})

			defaultAlias := GetPricingInfoForConfig(cfg, tt.provider, tt.model)
			if !defaultAlias.PricingUnavailable {
				t.Fatalf("configured default alias pricing = %#v, want unavailable", defaultAlias)
			}

			overrideAlias := GetPricingInfoForConfig(cfg, tt.provider, tt.model+"-override")
			if !overrideAlias.PricingUnavailable {
				t.Fatalf("configured override alias pricing = %#v, want unavailable", overrideAlias)
			}
		})
	}
}

func TestGetPricingInfoForConfig_ConfiguredKnownExactModelsAcrossProvidersStillPrice(t *testing.T) {
	tests := []struct {
		provider  string
		model     string
		wantInput float64
		wantOut   float64
	}{
		{provider: "claude", model: "claude-sonnet-4-6", wantInput: 3.00, wantOut: 15.00},
		{provider: "deepseek", model: "deepseek-v4-pro", wantInput: 1.74, wantOut: 3.48},
		{provider: "gemini", model: "gemini-3.1-pro", wantInput: 2.00, wantOut: 12.00},
		{provider: "groq", model: "llama-3.1-70b", wantInput: 0.59, wantOut: 0.79},
		{provider: "bedrock", model: "global.anthropic.claude-sonnet-4-6", wantInput: 3.00, wantOut: 15.00},
		{provider: "bedrock", model: "us.anthropic.claude-sonnet-4-6", wantInput: 3.30, wantOut: 16.50},
		{provider: "bedrock", model: "eu.anthropic.claude-sonnet-4-6", wantInput: 3.30, wantOut: 16.50},
		{provider: "bedrock", model: "au.anthropic.claude-sonnet-4-6", wantInput: 3.30, wantOut: 16.50},
		{provider: "bedrock", model: "amazon.nova-pro-v1:0", wantInput: 0.80, wantOut: 3.20},
		{provider: "bedrock", model: "meta.llama4-maverick-17b-instruct-v1:0", wantInput: 0.24, wantOut: 0.97},
		{provider: "bedrock", model: "global.anthropic.claude-sonnet-4-6-v1", wantInput: 3.00, wantOut: 15.00},
		{provider: "bedrock", model: "global.anthropic.claude-sonnet-4-6-v1:0", wantInput: 3.00, wantOut: 15.00},
	}

	for _, tt := range tests {
		t.Run(tt.provider+"/"+tt.model, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.SetProviderModelConfig(tt.provider, config.ProviderModelConfig{
				DefaultModel: tt.model,
			})

			got := GetPricingInfoForConfig(cfg, tt.provider, tt.model)
			if got.PricingUnavailable {
				t.Fatalf("configured known exact pricing = %#v, want available", got)
			}
			if got.InputCostPerM != tt.wantInput || got.OutputCostPerM != tt.wantOut {
				t.Fatalf("configured known exact pricing = %#v, want input=%f output=%f", got, tt.wantInput, tt.wantOut)
			}
		})
	}
}

func TestGetPricingInfoForConfig_ConfiguredPricingKnownExactModelWithoutCatalogStillPrices(t *testing.T) {
	tests := []struct {
		model     string
		wantInput float64
		wantOut   float64
	}{
		{model: "gpt-5.5", wantInput: 5.00, wantOut: 30.00},
		{model: "gpt-5.3-codex", wantInput: 1.75, wantOut: 14.00},
		{model: "gpt-5.2-codex", wantInput: 1.75, wantOut: 14.00},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.SetProviderModelConfig("openai", config.ProviderModelConfig{
				DefaultModel: tt.model,
			})

			got := GetPricingInfoForConfig(cfg, "openai", tt.model)
			if got.PricingUnavailable {
				t.Fatalf("configured known model pricing = %#v, want available", got)
			}
			if got.InputCostPerM != tt.wantInput || got.OutputCostPerM != tt.wantOut {
				t.Fatalf("configured known model pricing = %#v, want input=%f output=%f", got, tt.wantInput, tt.wantOut)
			}
		})
	}
}

func TestGetPricingInfoForConfig_ConfiguredPricingRuleMatchWithoutKnownExactIsUnavailable(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("openai", config.ProviderModelConfig{
		DefaultModel: "gpt-5.3",
	})

	got := GetPricingInfoForConfig(cfg, "openai", "gpt-5.3")
	if !got.PricingUnavailable {
		t.Fatalf("configured non-exact model pricing = %#v, want unavailable", got)
	}
}

func TestGetPricingInfoForConfig_ConfiguredOpenRouterModelIDWithoutCatalogStillPrices(t *testing.T) {
	tests := []struct {
		name      string
		model     string
		wantInput float64
		wantOut   float64
	}{
		{name: "anthropic id", model: "anthropic/claude-sonnet-4.6", wantInput: 3.00, wantOut: 15.00},
		{name: "openai id", model: "openai/gpt-5.4", wantInput: 2.50, wantOut: 15.00},
		{name: "static id", model: "zhipu/glm-5", wantInput: 0.72, wantOut: 2.30},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.SetProviderModelConfig("openrouter", config.ProviderModelConfig{
				DefaultModel: tt.model,
			})

			got := GetPricingInfoForConfig(cfg, "openrouter", tt.model)
			if got.PricingUnavailable {
				t.Fatalf("configured OpenRouter model ID pricing = %#v, want available", got)
			}
			if got.InputCostPerM != tt.wantInput || got.OutputCostPerM != tt.wantOut {
				t.Fatalf("configured OpenRouter model ID pricing = %#v, want input=%f output=%f", got, tt.wantInput, tt.wantOut)
			}
		})
	}
}

func TestGetPricingInfoForConfig_OpenRouterAliasWithCatalogModelIDPrices(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.SetProviderModelConfig("openrouter", config.ProviderModelConfig{
		DefaultModel: "corp-openrouter-gpt",
		CatalogModel: "openai/gpt-5.4",
	})

	got := GetPricingInfoForConfig(cfg, "openrouter", "corp-openrouter-gpt")
	if got.PricingUnavailable {
		t.Fatalf("OpenRouter alias with catalog_model pricing = %#v, want available", got)
	}
	if got.InputCostPerM != 2.50 || got.OutputCostPerM != 15.00 {
		t.Fatalf("OpenRouter alias with catalog_model pricing = %#v, want GPT-5.4 pricing", got)
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
			input: 1000000, output: 1000000, wantMin: 0.41, wantMax: 0.43,
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

func TestCalculateRequestCostWithCache_OpenAIGPT55CachedTokensDoNotTriggerLongInputTier(t *testing.T) {
	cost := CalculateRequestCostWithCache("openai", "gpt-5.5", api.Usage{
		InputTokens:       200000,
		CachedInputTokens: 100000,
	})

	// OpenAI cached_tokens は input_tokens の内訳なので、tier 判定は 200K のまま。
	// uncached: 100K * $5.00 = $0.50, cached: 100K * $0.50 = $0.05
	assertCostApprox(t, cost, 0.55)
}

func TestCalculateRequestCostWithCache_OpenAIGPT55LongInputUsesInputTokensOnly(t *testing.T) {
	cost := CalculateRequestCostWithCache("openai", "gpt-5.5", api.Usage{
		InputTokens:       300000,
		CachedInputTokens: 100000,
	})

	// input_tokens 自体が 272K を超えた場合だけ GPT-5.5 long_input tier になる。
	// uncached: 200K * $10.00 = $2.00, cached: 100K * $1.00 = $0.10
	assertCostApprox(t, cost, 2.10)
}

func TestCalculateRequestCostWithCache_OpenRouterDelegatedOpenAICachedTokensDoNotTriggerLongInputTier(t *testing.T) {
	cost := CalculateRequestCostWithCache("openrouter", "openai/gpt-5.5", api.Usage{
		InputTokens:       200000,
		CachedInputTokens: 100000,
	})

	assertCostApprox(t, cost, 0.55)
}

func TestCalculateRequestCostWithCache_ClaudeStillUsesCacheTokensForLongInputTier(t *testing.T) {
	cost := CalculateRequestCostWithCache("claude", "claude-sonnet-4-5", api.Usage{
		InputTokens:         150000,
		CachedInputTokens:   50000,
		CacheCreationTokens: 10000,
	})

	// Anthropic は input + cache_read + cache_creation の合計で 200K tier 判定する。
	// uncached: 90K * $6.00 = $0.54, cached: 50K * $0.60 = $0.03, creation: 10K * $7.50 = $0.075
	assertCostApprox(t, cost, 0.645)
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

func TestEstimateRequestCostWithCache_UnknownPricing(t *testing.T) {
	estimate := EstimateRequestCostWithCache("bedrock", "amazon.nova-unknown-v1:0", api.Usage{
		InputTokens:  1000000,
		OutputTokens: 1000000,
	})
	if !estimate.PricingUnavailable {
		t.Fatalf("PricingUnavailable = false, want true: %#v", estimate)
	}
	if estimate.Cost != 0 {
		t.Fatalf("Cost = %f, want 0 for unavailable pricing", estimate.Cost)
	}
}

func assertCostApprox(t *testing.T, got, want float64) {
	t.Helper()
	const tolerance = 0.000001
	if got < want-tolerance || got > want+tolerance {
		t.Fatalf("cost = %f, want %f", got, want)
	}
}
