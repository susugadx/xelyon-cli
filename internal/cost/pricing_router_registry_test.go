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

func TestPricingResolverRegistry_UnknownFamilyUnavailable(t *testing.T) {
	got := resolvePricingByFamily("unknown-family", pricingRequest{
		Model: "test-model",
	})
	if !got.PricingUnavailable {
		t.Fatalf("resolvePricingByFamily(unknown).PricingUnavailable = false, want true: %#v", got)
	}
}
