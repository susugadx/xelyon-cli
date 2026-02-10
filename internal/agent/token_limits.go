package agent

import "github.com/susugadx/xelyon-cli/internal/agent/token"

// GetTokenUsagePercentage はトークン使用率を計算
func (a *Agent) GetTokenUsagePercentage() float64 {
	currentTokens := a.EstimateTokens()
	limit := token.GetModelTokenLimit(a.CurrentModel)
	if limit == 0 {
		return 0
	}
	return float64(currentTokens) / float64(limit) * 100
}

// EstimateTokens は現在のトークン使用量を推定
func (a *Agent) EstimateTokens() int {
	total := 0

	// システムプロンプト
	total += token.EstimateTokenCount(a.SystemPrompt)

	// 会話履歴
	for _, msg := range a.History {
		total += token.EstimateTokenCount(msg.Content)
	}

	// FC プロバイダーはツール定義を JSON で別送信 → トークン消費に含める
	// （非FC プロバイダーはシステムプロンプト内にツール説明を含むため二重計上を避ける）
	if a.CurrentProvider != nil && a.CurrentProvider.IsFunctionCallingEnabled() {
		total += estimateToolDefinitionTokens()
	}

	return total
}

// EstimateSystemPromptTokens はシステムプロンプトのトークン数を推定
func (a *Agent) EstimateSystemPromptTokens() int {
	return token.EstimateTokenCount(a.SystemPrompt)
}

// EstimateHistoryTokens は会話履歴のトークン数を推定
func (a *Agent) EstimateHistoryTokens() int {
	total := 0
	for _, msg := range a.History {
		total += token.EstimateTokenCount(msg.Content)
	}
	return total
}
