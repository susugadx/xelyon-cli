package agent

import (
	_ "embed"
	"sync"

	"gopkg.in/yaml.v3"
)

//go:embed pricing.yaml
var embeddedPricingYAML []byte

var (
	pricingConfigOnce   sync.Once
	loadedPricingConfig *pricingConfig
)

func loadPricingConfig() *pricingConfig {
	pricingConfigOnce.Do(func() {
		if len(embeddedPricingYAML) == 0 {
			return
		}

		var cfg pricingConfig
		if err := yaml.Unmarshal(embeddedPricingYAML, &cfg); err != nil {
			return
		}
		loadedPricingConfig = &cfg
	})
	return loadedPricingConfig
}
