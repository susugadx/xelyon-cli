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
		return 50000 // 64K物理上限前
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
