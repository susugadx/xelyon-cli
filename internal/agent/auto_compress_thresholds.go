package agent

import "strings"

// GetProviderCompressThreshold はプロバイダとモデルに基づく
// コスト最適化のための絶対トークン閾値を返す。
// 0 を返した場合は絶対値閾値なし（既存の%ベースを使用）。
func GetProviderCompressThreshold(provider string, model string) int {
	switch provider {
	case "gemini":
		return 180000 // 200K pricing cliff回避
	case "claude", "bedrock":
		return 150000 // 200K pricing cliff回避
	case "deepseek":
		return 50000 // 128K物理上限より十分手前
	case "openai":
		lm := strings.ToLower(model)
		if strings.Contains(lm, "5.4") {
			return 260000 // 272K pricing cliff回避（2x課金ライン手前）
		}
		return 100000 // キャッシュコスト削減
	case "openrouter":
		return 120000 // モデル依存だが安全な値
	default:
		return 0 // 不明なプロバイダは既存ロジックに任せる
	}
}

func averageOutputTokens(stats *SessionStats) int {
	if stats == nil || stats.OutputTokens <= 0 {
		return 0
	}
	assistantMessages := stats.AssistantMessages
	if assistantMessages < 1 {
		assistantMessages = 1
	}
	return stats.OutputTokens / assistantMessages
}

func shouldForceCompressForPricingCliff(provider, model string, currentTokens int, stats *SessionStats) (int, bool) {
	if currentTokens <= 0 {
		return currentTokens, false
	}

	projectedTokens := currentTokens + averageOutputTokens(stats)
	currentPricing := GetPricingInfo(provider, model, currentTokens)
	projectedPricing := GetPricingInfo(provider, model, projectedTokens)
	if projectedPricing.InputCostPerM > currentPricing.InputCostPerM {
		return projectedTokens, true
	}

	return projectedTokens, false
}
