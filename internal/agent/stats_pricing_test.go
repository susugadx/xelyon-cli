package agent

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
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

// --- FormatTokens tests ---

func TestFormatTokens_AllRanges(t *testing.T) {
	tests := []struct {
		input int
		want  string
	}{
		{0, "0"},
		{999, "999"},
		{1500, "1.5k"},
		{1500000, "1.5M"},
		{100, "100"},
		{10000, "10.0k"},
	}

	for _, tt := range tests {
		got := FormatTokens(tt.input)
		if got != tt.want {
			t.Errorf("FormatTokens(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// --- FormatNumber tests ---

func TestFormatNumber_AllRanges(t *testing.T) {
	tests := []struct {
		input int
		want  string
	}{
		{0, "0"},
		{42, "42"},
		{999, "999"},
		{1234, "1,234"},
		{10000, "10,000"},
		{1000000, "1,000,000"},
		{1234567, "1,234,567"},
	}

	for _, tt := range tests {
		got := FormatNumber(tt.input)
		if got != tt.want {
			t.Errorf("FormatNumber(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// --- FormatFileSize tests ---

func TestFormatFileSize_AllRanges(t *testing.T) {
	tests := []struct {
		input int64
		want  string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1023, "1023 B"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
	}

	for _, tt := range tests {
		got := FormatFileSize(tt.input)
		if got != tt.want {
			t.Errorf("FormatFileSize(%d) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// --- sessionCacheHitRate tests ---

func TestSessionCacheHitRate_Various(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		cached   int
		wantRate float64
	}{
		{name: "zero input", input: 0, cached: 0, wantRate: 0},
		{name: "negative input", input: -1, cached: 0, wantRate: 0},
		{name: "50% hit", input: 200, cached: 100, wantRate: 50.0},
		{name: "100% hit", input: 100, cached: 100, wantRate: 100.0},
		{name: "0% hit", input: 100, cached: 0, wantRate: 0},
		{name: "25% hit", input: 200, cached: 50, wantRate: 25.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := &SessionStats{
				InputTokens:       tt.input,
				CachedInputTokens: tt.cached,
			}
			got := sessionCacheHitRate(stats)
			if got != tt.wantRate {
				t.Errorf("sessionCacheHitRate() = %f, want %f", got, tt.wantRate)
			}
		})
	}
}

// --- NewSessionStats tests ---

func TestNewSessionStats_WithModel(t *testing.T) {
	stats := NewSessionStats("claude", "claude-opus-4-6")
	if stats.Provider != "claude" {
		t.Errorf("Provider = %q, want %q", stats.Provider, "claude")
	}
	if stats.Model != "claude-opus-4-6" {
		t.Errorf("Model = %q, want %q", stats.Model, "claude-opus-4-6")
	}
	if stats.ToolExecutions == nil {
		t.Error("ToolExecutions should be initialized")
	}
	if stats.StartTime.IsZero() {
		t.Error("StartTime should be set")
	}
}

func TestNewSessionStats_WithoutModel(t *testing.T) {
	stats := NewSessionStats("openai")
	if stats.Model != "" {
		t.Errorf("Model = %q, want empty", stats.Model)
	}
}

// --- SavingsMetrics tests ---

func TestSavingsMetrics_AddAndHasAny(t *testing.T) {
	m := SavingsMetrics{}
	if m.hasAny() {
		t.Error("empty SavingsMetrics.hasAny() should be false")
	}

	m.add(SavingsMetrics{SavedCalls: 1})
	if !m.hasAny() {
		t.Error("SavingsMetrics with SavedCalls=1 should hasAny()")
	}
	if m.SavedCalls != 1 {
		t.Errorf("SavedCalls = %d, want 1", m.SavedCalls)
	}

	m.add(SavingsMetrics{EstimatedInputTokensSaved: 500, EstimatedCostSaved: 0.05})
	if m.EstimatedInputTokensSaved != 500 {
		t.Errorf("EstimatedInputTokensSaved = %d, want 500", m.EstimatedInputTokensSaved)
	}
	if m.EstimatedCostSaved < 0.049 || m.EstimatedCostSaved > 0.051 {
		t.Errorf("EstimatedCostSaved = %f, want ~0.05", m.EstimatedCostSaved)
	}
}

func TestSavingsMetrics_HasAny_EachField(t *testing.T) {
	tests := []struct {
		name    string
		metrics SavingsMetrics
		want    bool
	}{
		{name: "empty", metrics: SavingsMetrics{}, want: false},
		{name: "saved calls only", metrics: SavingsMetrics{SavedCalls: 1}, want: true},
		{name: "tokens saved only", metrics: SavingsMetrics{EstimatedInputTokensSaved: 1}, want: true},
		{name: "cost saved only", metrics: SavingsMetrics{EstimatedCostSaved: 0.01}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.metrics.hasAny(); got != tt.want {
				t.Errorf("hasAny() = %v, want %v", got, tt.want)
			}
		})
	}
}

// --- OptimizationMetrics tests ---

func TestOptimizationMetrics_AddAndHasAny(t *testing.T) {
	m := OptimizationMetrics{}
	if m.hasAny() {
		t.Error("empty OptimizationMetrics.hasAny() should be false")
	}

	m.add(OptimizationMetrics{NegativeCacheHits: 3, CompactionCount: 2})
	if !m.hasAny() {
		t.Error("OptimizationMetrics with values should hasAny()")
	}
	if m.NegativeCacheHits != 3 {
		t.Errorf("NegativeCacheHits = %d, want 3", m.NegativeCacheHits)
	}
	if m.CompactionCount != 2 {
		t.Errorf("CompactionCount = %d, want 2", m.CompactionCount)
	}
}

func TestOptimizationMetrics_HasAny_EachField(t *testing.T) {
	tests := []struct {
		name    string
		metrics OptimizationMetrics
		want    bool
	}{
		{name: "empty", metrics: OptimizationMetrics{}, want: false},
		{name: "NegativeCacheHits", metrics: OptimizationMetrics{NegativeCacheHits: 1}, want: true},
		{name: "ErrorCompressions", metrics: OptimizationMetrics{ErrorCompressions: 1}, want: true},
		{name: "FailedPairCompressions", metrics: OptimizationMetrics{FailedPairCompressions: 1}, want: true},
		{name: "OutlineFirstCount", metrics: OptimizationMetrics{OutlineFirstCount: 1}, want: true},
		{name: "CompactionCount", metrics: OptimizationMetrics{CompactionCount: 1}, want: true},
		{name: "CostAwareCompressions", metrics: OptimizationMetrics{CostAwareCompressions: 1}, want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.metrics.hasAny(); got != tt.want {
				t.Errorf("hasAny() = %v, want %v", got, tt.want)
			}
		})
	}
}
