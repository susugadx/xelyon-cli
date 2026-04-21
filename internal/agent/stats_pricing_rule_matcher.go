package agent

import "strings"

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
