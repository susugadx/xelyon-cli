package agent

// PricingInfo はプロバイダー別の料金情報（$/1M tokens）
type PricingInfo struct {
	InputCostPerM         float64 // 通常入力トークン
	OutputCostPerM        float64 // 出力トークン
	CachedInputCostPerM   float64 // キャッシュヒット入力（割引後）
	CacheCreationCostPerM float64 // キャッシュ作成（Claude: 1.25x）
}

type pricingRule struct {
	Contains  []string       `yaml:"contains"`
	Pricing   PricingInfo    `yaml:"pricing"`
	LongInput *longInputTier `yaml:"long_input"` // ルール別 long context ティア
}

type providerPricingConfig struct {
	Default   PricingInfo    `yaml:"default"`
	LongInput *longInputTier `yaml:"long_input"`
	Rules     []pricingRule  `yaml:"rules"`
}

type longInputTier struct {
	Threshold int         `yaml:"threshold"`
	Pricing   PricingInfo `yaml:"pricing"`
}

type pricingConfig struct {
	OpenAI   providerPricingConfig `yaml:"openai"`
	Claude   providerPricingConfig `yaml:"claude"`
	Gemini   providerPricingConfig `yaml:"gemini"`
	DeepSeek providerPricingConfig `yaml:"deepseek"`
	Groq     providerPricingConfig `yaml:"groq"`
	Kimi     providerPricingConfig `yaml:"kimi"`
}
