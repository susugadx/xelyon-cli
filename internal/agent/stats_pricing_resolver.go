package agent

import (
	_ "embed"
	"strings"
	"sync"

	"github.com/susugadx/xelyon-cli/internal/config"
	"gopkg.in/yaml.v3"
)

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

func matchPricingRules(lm string, provider providerPricingConfig, promptTokenCount int) (PricingInfo, bool) {
	for _, rule := range provider.Rules {
		for _, keyword := range rule.Contains {
			if strings.Contains(lm, keyword) {
				// ルール別 long_input ティアがあれば優先チェック
				if rule.LongInput != nil && promptTokenCount > rule.LongInput.Threshold {
					return rule.LongInput.Pricing, true
				}
				return rule.Pricing, true
			}
		}
	}
	return PricingInfo{}, false
}

// GetPricingInfo はプロバイダー・モデル別の料金情報を返す
// promptTokenCount はオプション（Gemini 200Kティア判定用）
func GetPricingInfo(provider string, model string, promptTokenCount ...int) PricingInfo {
	ptc := 0
	if len(promptTokenCount) > 0 {
		ptc = promptTokenCount[0]
	}
	switch config.CanonicalProviderName(provider) {
	case "deepseek":
		// DeepSeekの料金体系
		return getDeepSeekPricing(model)
	case "openai":
		return getOpenAIPricing(model, ptc)
	case "claude":
		return getClaudePricing(model, ptc)
	case "bedrock":
		if strings.Contains(strings.ToLower(model), "claude") {
			return getClaudePricing(model, ptc)
		}
		// Claude以外のBedrockモデルは一旦汎用料金
		return PricingInfo{
			InputCostPerM:         0.28,
			OutputCostPerM:        0.42,
			CachedInputCostPerM:   0.028,
			CacheCreationCostPerM: 0.28,
		}
	case "gemini":
		return getGeminiPricing(model, ptc)
	case "groq":
		return getGroqPricing(model)
	case "openrouter":
		return getOpenRouterPricing(model, ptc)
	case "ollama":
		return PricingInfo{} // ローカル実行は無料
	default:
		// 不明なプロバイダーはDeepSeek V3.2料金で概算
		return PricingInfo{
			InputCostPerM:         0.28,
			OutputCostPerM:        0.42,
			CachedInputCostPerM:   0.028,
			CacheCreationCostPerM: 0.28,
		}
	}
}
