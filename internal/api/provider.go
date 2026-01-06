package api

import (
	"context"
)

// Provider はLLMプロバイダーの共通インターフェース
type Provider interface {
	// Name はプロバイダー名を返す
	Name() string

	// ChatWithTools はツール対応の会話を行う（ストリーミング）
	ChatWithTools(ctx context.Context, systemPrompt string, history []Message, model string) (string, error)
}
