package cost

// PricingInfo はプロバイダー別の料金情報（$/1M tokens）
type PricingInfo struct {
	InputCostPerM         float64 // 通常入力トークン
	OutputCostPerM        float64 // 出力トークン
	CachedInputCostPerM   float64 // キャッシュヒット入力（割引後）
	CacheCreationCostPerM float64 // キャッシュ作成（Claude: 1.25x）
	PricingUnavailable    bool    // 既知の料金表にないためUSD見積もり不可
}

func pricingUnavailableInfo() PricingInfo {
	return PricingInfo{PricingUnavailable: true}
}

type pricingRule struct {
	Contains  []string       `yaml:"contains"`
	Pricing   PricingInfo    `yaml:"pricing"`
	LongInput *longInputTier `yaml:"long_input"` // ルール別 long context ティア
}

type pricingKnownModels struct {
	Exact []string `yaml:"exact"`
}

type providerPricingConfig struct {
	KnownModels pricingKnownModels `yaml:"known_models"`
	Default     PricingInfo        `yaml:"default"`
	LongInput   *longInputTier     `yaml:"long_input"`
	Rules       []pricingRule      `yaml:"rules"`
}

type longInputTier struct {
	Threshold int         `yaml:"threshold"`
	Pricing   PricingInfo `yaml:"pricing"`
}

type pricingConfig map[string]providerPricingConfig

func (c pricingConfig) provider(family string) (providerPricingConfig, bool) {
	if c == nil {
		return providerPricingConfig{}, false
	}
	provider, ok := c[family]
	return provider, ok
}
