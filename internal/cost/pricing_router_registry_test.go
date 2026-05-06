package cost

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/llmcatalog"
)

func TestPricingResolverRegistry_CoversCatalogFamilies(t *testing.T) {
	for _, provider := range llmcatalog.ProviderKeys(false) {
		entry, ok := llmcatalog.ProviderDescriptorFor(provider)
		if !ok {
			t.Fatalf("missing descriptor for provider %q", provider)
		}
		if entry.PricingFamily == "" {
			t.Fatalf("provider %q has empty PricingFamily", provider)
		}
		if _, ok := pricingResolvers[entry.PricingFamily]; !ok {
			t.Fatalf("pricing resolver missing for provider %q family %q", provider, entry.PricingFamily)
		}
	}
}

func TestPricingResolverRegistry_CoversLoadedPricingFamilies(t *testing.T) {
	cfg := loadPricingConfig()
	if cfg == nil {
		t.Fatal("loadPricingConfig() = nil")
	}

	for family := range *cfg {
		if _, ok := pricingResolvers[family]; !ok {
			t.Fatalf("pricing resolver missing for loaded pricing family %q", family)
		}
	}
}

func TestPricingResolverRegistry_KimiFamilyReadyForFutureProviderDescriptor(t *testing.T) {
	tests := []struct {
		model      string
		wantInput  float64
		wantOutput float64
		wantCached float64
	}{
		{model: "kimi-k2.6", wantInput: 0.95, wantOutput: 4.00, wantCached: 0.16},
		{model: "kimi-k2.5", wantInput: 0.60, wantOutput: 3.00, wantCached: 0.10},
	}

	for _, tt := range tests {
		t.Run(tt.model, func(t *testing.T) {
			got := resolvePricingByFamily("kimi", pricingRequest{
				Model: tt.model,
			})
			if got.PricingUnavailable {
				t.Fatalf("resolvePricingByFamily(kimi).PricingUnavailable = true, want false: %#v", got)
			}
			if got.InputCostPerM != tt.wantInput || got.OutputCostPerM != tt.wantOutput || got.CachedInputCostPerM != tt.wantCached {
				t.Fatalf("resolvePricingByFamily(kimi) = %#v, want input=%f output=%f cached=%f", got, tt.wantInput, tt.wantOutput, tt.wantCached)
			}
		})
	}
}

func TestPricingResolverRegistry_UnknownFamilyUnavailable(t *testing.T) {
	got := resolvePricingByFamily("unknown-family", pricingRequest{
		Model: "test-model",
	})
	if !got.PricingUnavailable {
		t.Fatalf("resolvePricingByFamily(unknown).PricingUnavailable = false, want true: %#v", got)
	}
}
