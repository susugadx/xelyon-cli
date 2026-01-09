package api

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"time"
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

// handleRateLimit はレート制限エラーを処理
func handleRateLimit(resp *http.Response) error {
	if resp.StatusCode != 429 {
		return nil // レート制限エラーではない
	}

	// Retry-Afterヘッダーをチェック
	retryAfter := resp.Header.Get("Retry-After")
	if retryAfter == "" {
		// ヘッダーがない場合はデフォルト待機時間
		return fmt.Errorf("rate limit exceeded (429). Please retry after 60 seconds")
	}

	// Retry-Afterは秒数またはHTTP-date形式
	// まず秒数として解釈を試みる
	if seconds, err := strconv.Atoi(retryAfter); err == nil {
		return fmt.Errorf("rate limit exceeded (429). Please retry after %d seconds", seconds)
	}

	// HTTP-date形式の場合
	if retryTime, err := http.ParseTime(retryAfter); err == nil {
		waitDuration := time.Until(retryTime)
		if waitDuration > 0 {
			return fmt.Errorf("rate limit exceeded (429). Please retry after %v", waitDuration.Round(time.Second))
		}
	}

	return fmt.Errorf("rate limit exceeded (429). Please retry later")
}
