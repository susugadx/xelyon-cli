package cost

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func TestGetGeminiPricing_AllModels(t *testing.T) {
	tests := []struct {
		name        string
		model       string
		ptc         int
		wantInput   float64
		wantOutput  float64
		wantStorage float64
	}{
		{name: "2.5-pro normal", model: "gemini-2.5-pro", ptc: 100000, wantInput: 1.25, wantOutput: 10.00, wantStorage: 4.50},
		{name: "2.5-pro long", model: "gemini-2.5-pro", ptc: 250000, wantInput: 2.50, wantOutput: 15.00, wantStorage: 4.50},
		{name: "2.5-flash", model: "gemini-2.5-flash", ptc: 0, wantInput: 0.30, wantOutput: 2.50, wantStorage: 1.00},
		{name: "3.5-flash", model: "gemini-3.5-flash", ptc: 0, wantInput: 1.50, wantOutput: 9.00, wantStorage: 1.00},
		{name: "3.1-flash-lite", model: "gemini-3.1-flash-lite", ptc: 0, wantInput: 0.25, wantOutput: 1.50, wantStorage: 1.00},
		{name: "3.x flash default", model: "gemini-3-flash", ptc: 0, wantInput: 0.50, wantOutput: 3.00, wantStorage: 1.00},
		{name: "3.1-pro normal", model: "gemini-3.1-pro", ptc: 100000, wantInput: 2.00, wantOutput: 12.00, wantStorage: 4.50},
		{name: "3.1-pro long", model: "gemini-3.1-pro", ptc: 250000, wantInput: 4.00, wantOutput: 18.00, wantStorage: 4.50},
		{name: "2.0-flash-lite no storage", model: "gemini-2.0-flash-lite", ptc: 0, wantInput: 0.075, wantOutput: 0.30, wantStorage: 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pricing := getGeminiStandardPricing(tt.model, tt.ptc)
			if pricing.InputCostPerM != tt.wantInput {
				t.Errorf("InputCostPerM = %f, want %f", pricing.InputCostPerM, tt.wantInput)
			}
			if pricing.OutputCostPerM != tt.wantOutput {
				t.Errorf("OutputCostPerM = %f, want %f", pricing.OutputCostPerM, tt.wantOutput)
			}
			if pricing.CacheStorageCostPerMHour != tt.wantStorage {
				t.Errorf("CacheStorageCostPerMHour = %f, want %f", pricing.CacheStorageCostPerMHour, tt.wantStorage)
			}
		})
	}
}

func TestGetPricingInfoForConfig_GeminiServiceTier(t *testing.T) {
	flexCfg := config.DefaultConfig()
	flexCfg.Gemini.ServiceTier = config.GeminiServiceTierFlex

	flexLite := GetPricingInfoForConfig(flexCfg, "gemini", "gemini-3.1-flash-lite")
	if flexLite.PricingUnavailable {
		t.Fatalf("flex flash-lite pricing unavailable: %#v", flexLite)
	}
	if flexLite.InputCostPerM != 0.125 || flexLite.OutputCostPerM != 0.75 || flexLite.CacheStorageCostPerMHour != 0.50 {
		t.Fatalf("flex flash-lite pricing = %#v, want flex rates", flexLite)
	}

	priorityCfg := config.DefaultConfig()
	priorityCfg.Gemini.ServiceTier = config.GeminiServiceTierPriority

	priorityProLong := GetPricingInfoForConfig(priorityCfg, "gemini", "gemini-3.1-pro-preview", 250000)
	if priorityProLong.PricingUnavailable {
		t.Fatalf("priority pro long pricing unavailable: %#v", priorityProLong)
	}
	if priorityProLong.InputCostPerM != 7.20 || priorityProLong.OutputCostPerM != 32.40 || priorityProLong.CacheStorageCostPerMHour != 8.10 {
		t.Fatalf("priority pro long pricing = %#v, want priority long-context rates", priorityProLong)
	}

	standard := GetPricingInfoForConfig(config.DefaultConfig(), "gemini", "gemini-3.1-flash-lite")
	if standard.InputCostPerM != 0.25 || standard.OutputCostPerM != 1.50 || standard.CacheStorageCostPerMHour != 1.00 {
		t.Fatalf("standard flash-lite pricing = %#v, want standard rates", standard)
	}
}

func TestGetPricingInfoForConfig_GeminiServiceTierPreviewModels(t *testing.T) {
	tests := []struct {
		name       string
		tier       string
		model      string
		wantInput  float64
		wantOutput float64
	}{
		{
			name:       "flex Gemini 3 Pro preview",
			tier:       config.GeminiServiceTierFlex,
			model:      "gemini-3-pro-preview",
			wantInput:  1.00,
			wantOutput: 6.00,
		},
		{
			name:       "priority Gemini 3 Pro preview",
			tier:       config.GeminiServiceTierPriority,
			model:      "gemini-3-pro-preview",
			wantInput:  3.60,
			wantOutput: 21.60,
		},
		{
			name:       "flex Gemini 2.5 Pro preview",
			tier:       config.GeminiServiceTierFlex,
			model:      "gemini-2.5-pro-preview",
			wantInput:  0.625,
			wantOutput: 5.00,
		},
		{
			name:       "priority Gemini 2.5 Pro preview",
			tier:       config.GeminiServiceTierPriority,
			model:      "gemini-2.5-pro-preview",
			wantInput:  2.25,
			wantOutput: 18.00,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.Gemini.ServiceTier = tt.tier

			got := GetPricingInfoForConfig(cfg, "gemini", tt.model)
			if got.PricingUnavailable {
				t.Fatalf("GetPricingInfoForConfig(%q, %q) PricingUnavailable = true, want false", tt.tier, tt.model)
			}
			if got.InputCostPerM != tt.wantInput || got.OutputCostPerM != tt.wantOutput {
				t.Fatalf("GetPricingInfoForConfig(%q, %q) = %#v, want input=%f output=%f", tt.tier, tt.model, got, tt.wantInput, tt.wantOutput)
			}
		})
	}
}

func TestGetPricingInfoForConfig_GeminiServiceTierUsesCatalogModel(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Gemini.ServiceTier = config.GeminiServiceTierFlex
	cfg.SetProviderModelConfig("gemini", config.ProviderModelConfig{
		DefaultModel: "corp-gemini-flash-lite",
		CatalogModel: "models/gemini-3.1-flash-lite",
	})

	got := GetPricingInfoForConfig(cfg, "gemini", "corp-gemini-flash-lite")
	if got.PricingUnavailable {
		t.Fatalf("alias pricing unavailable: %#v", got)
	}
	if got.InputCostPerM != 0.125 || got.OutputCostPerM != 0.75 || got.CacheStorageCostPerMHour != 0.50 {
		t.Fatalf("alias pricing = %#v, want catalog model flex rates", got)
	}
}

func TestEstimateRequestCostWithCacheForConfig_GeminiBillingServiceTierOverride(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Gemini.ServiceTier = config.GeminiServiceTierPriority

	usage := api.Usage{
		InputTokens:        1000000,
		OutputTokens:       1000000,
		BillingServiceTier: config.GeminiServiceTierStandard,
	}
	estimate := EstimateRequestCostWithCacheForConfig(cfg, "gemini", "gemini-3.1-flash-lite", usage)
	if estimate.PricingUnavailable {
		t.Fatalf("PricingUnavailable = true, want false")
	}
	assertCostApprox(t, estimate.Cost, 1.75)
}

func TestEstimateCacheStorageCost_GeminiModelRates(t *testing.T) {
	tests := []struct {
		name       string
		model      string
		tokens     int
		ttlSeconds int
		want       float64
	}{
		{
			name:       "flash lite standard",
			model:      "gemini-3.1-flash-lite",
			tokens:     1000000,
			ttlSeconds: 3600,
			want:       1.00,
		},
		{
			name:       "pro long tier keeps standard storage rate",
			model:      "gemini-3.1-pro-preview",
			tokens:     250000,
			ttlSeconds: 3600,
			want:       1.125,
		},
		{
			name:       "pro ttl scales by hour",
			model:      "gemini-3.1-pro-preview",
			tokens:     1000000,
			ttlSeconds: 7200,
			want:       9.00,
		},
		{
			name:       "2.0 flash lite has no explicit cache storage pricing",
			model:      "gemini-2.0-flash-lite",
			tokens:     1000000,
			ttlSeconds: 3600,
			want:       0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			estimate := EstimateCacheStorageCost("gemini", tt.model, tt.tokens, tt.ttlSeconds)
			if estimate.PricingUnavailable {
				t.Fatalf("PricingUnavailable = true, want false")
			}
			assertCostApprox(t, estimate.Cost, tt.want)
		})
	}
}

func TestEstimateCacheStorageCost_GeminiServiceTierRates(t *testing.T) {
	flexCfg := config.DefaultConfig()
	flexCfg.Gemini.ServiceTier = config.GeminiServiceTierFlex

	flex := EstimateCacheStorageCostForConfig(flexCfg, "gemini", "gemini-3.1-flash-lite", 1000000, 3600)
	if flex.PricingUnavailable {
		t.Fatalf("flex PricingUnavailable = true, want false")
	}
	assertCostApprox(t, flex.Cost, 0.50)

	priorityCfg := config.DefaultConfig()
	priorityCfg.Gemini.ServiceTier = config.GeminiServiceTierPriority

	priority := EstimateCacheStorageCostForConfig(priorityCfg, "gemini", "gemini-3.1-flash-lite", 1000000, 3600)
	if priority.PricingUnavailable {
		t.Fatalf("priority PricingUnavailable = true, want false")
	}
	assertCostApprox(t, priority.Cost, 1.80)
}

func TestEstimateCacheStorageCost_GeminiUnknownPricing(t *testing.T) {
	estimate := EstimateCacheStorageCost("gemini", "gemini-unknown", 1000000, 3600)
	if !estimate.PricingUnavailable {
		t.Fatalf("PricingUnavailable = false, want true: %#v", estimate)
	}
	if estimate.Cost != 0 {
		t.Fatalf("Cost = %f, want 0 for unavailable pricing", estimate.Cost)
	}
}
