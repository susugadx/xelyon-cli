package agent

import (
	"github.com/susugadx/xelyon-cli/internal/agent/token"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func (a *Agent) currentModelTokenLimit(cfg *config.Config) int {
	if a == nil {
		return 0
	}
	if cfg == nil {
		cfg = a.cfg()
	}
	return token.GetModelTokenLimitForConfig(cfg, a.activeModelProviderConfigKey(cfg), a.CurrentModel)
}

// GetTokenUsagePercentage はトークン使用率を計算
func (a *Agent) GetTokenUsagePercentage() float64 {
	currentTokens := a.EstimateTokens()
	limit := a.currentModelTokenLimit(a.cfg())
	if limit == 0 {
		return 0
	}
	return float64(currentTokens) / float64(limit) * 100
}

// EstimateTokens は現在のトークン使用量を推定
func (a *Agent) EstimateTokens() int {
	total := 0

	// システムプロンプト
	total += token.EstimateTokenCountForModel(a.CurrentModel, a.SystemPrompt)

	// 会話履歴
	for _, msg := range a.History {
		total += token.EstimateTokenCountForModel(a.CurrentModel, msg.Content)
	}

	// FC プロバイダーはツール定義を JSON で別送信 → トークン消費に含める
	// （非FC プロバイダーはシステムプロンプト内にツール説明を含むため二重計上を避ける）
	if a.CurrentProvider != nil && a.CurrentProvider.IsFunctionCallingEnabled() {
		total += a.estimateToolDefinitionTokens()
	}

	return total
}

// EstimateSystemPromptTokens はシステムプロンプトのトークン数を推定
func (a *Agent) EstimateSystemPromptTokens() int {
	return token.EstimateTokenCountForModel(a.CurrentModel, a.SystemPrompt)
}

// EstimateHistoryTokens は会話履歴のトークン数を推定
func (a *Agent) EstimateHistoryTokens() int {
	total := 0
	for _, msg := range a.History {
		total += token.EstimateTokenCountForModel(a.CurrentModel, msg.Content)
	}
	return total
}
