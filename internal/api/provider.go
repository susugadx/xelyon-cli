package api

import (
	"context"
	"fmt"
	"regexp"
)

// Provider はLLMプロバイダーの共通インターフェース
type Provider interface {
	// Name はプロバイダー名を返す
	Name() string

	// ChatWithTools はツール対応の会話を行う（ストリーミング）
	ChatWithTools(ctx context.Context, systemPrompt string, history []Message, model string) (string, error)
}

// sanitizeErrorMessage はエラーメッセージから機密情報を削除
func sanitizeErrorMessage(body []byte, statusCode int) error {
	const maxLen = 200 // エラーメッセージの最大長

	if len(body) == 0 {
		return fmt.Errorf("API error (%d): empty response", statusCode)
	}

	message := string(body)

	// APIキーのパターンを削除（sk-, Bearer, api_key= など）
	apiKeyPatterns := []*regexp.Regexp{
		regexp.MustCompile(`sk-[a-zA-Z0-9]{20,}`),                 // OpenAI形式
		regexp.MustCompile(`Bearer [a-zA-Z0-9_\-\.]{20,}`),        // Bearer token
		regexp.MustCompile(`api_key[=:]\s*[a-zA-Z0-9_\-\.]{20,}`), // api_key=
		regexp.MustCompile(`"key":\s*"[a-zA-Z0-9_\-\.]{20,}"`),    // JSON key
		regexp.MustCompile(`AIza[a-zA-Z0-9_\-]{30,}`),             // Google API key
		regexp.MustCompile(`AKIA[A-Z0-9]{16}`),                    // AWS key
	}

	for _, pattern := range apiKeyPatterns {
		message = pattern.ReplaceAllString(message, "[REDACTED]")
	}

	// 長すぎる場合は切り詰め
	if len(message) > maxLen {
		message = message[:maxLen] + "... (truncated)"
	}

	return fmt.Errorf("API error (%d): %s", statusCode, message)
}
