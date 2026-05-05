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
	got := resolvePricingByFamily("kimi", pricingRequest{
		Model: "kimi-k2.5",
	})
	if got.PricingUnavailable {
		t.Fatalf("resolvePricingByFamily(kimi).PricingUnavailable = true, want false: %#v", got)
	}
	if got.InputCostPerM != 0.60 || got.OutputCostPerM != 3.00 || got.CachedInputCostPerM != 0.06 {
		t.Fatalf("resolvePricingByFamily(kimi) = %#v, want Kimi K2.5 pricing", got)
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
