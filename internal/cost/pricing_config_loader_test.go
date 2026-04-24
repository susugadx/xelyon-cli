package cost

import (
	"sync"
	"testing"
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
}
