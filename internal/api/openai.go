package api

import (
	"context"
	"net/http"
	"os"

	"github.com/susugadx/xelyon-cli/internal/config"
)

const defaultOpenAIURL = "https://api.openai.com/v1/chat/completions"

// OpenAIProvider はOpenAI APIのプロバイダー実装
type OpenAIProvider struct {
	apiKey     string
	apiURL     string
	httpClient *http.Client
}

// NewOpenAIProvider は新しいOpenAIProviderを作成
func NewOpenAIProvider(apiKey string) *OpenAIProvider {
	// 環境変数からURLをオーバーライド可能
	apiURL := os.Getenv("OPENAI_API_URL")
	if apiURL == "" {
		apiURL = defaultOpenAIURL
	}

	return &OpenAIProvider{
		apiKey: apiKey,
		apiURL: apiURL,
		httpClient: &http.Client{
			Timeout: config.DefaultHTTPTimeout,
		},
	}
}

// Name はプロバイダー名を返す
func (p *OpenAIProvider) Name() string {
	return "OpenAI"
}

// SupportsImages は画像入力対応を返す
func (p *OpenAIProvider) SupportsImages() bool {
	return true
}

// levelToReasoningEffort は Thinking Level を OpenAI reasoning_effort に変換
func levelToReasoningEffort(level string) string {
	switch level {
	case "low":
		return "low"
	case "medium":
		return "medium"
	case "high", "xhigh":
		return "high"
	default:
		return "medium"
	}
}

// ChatWithTools は Provider interface の実装（context対応）
func (p *OpenAIProvider) ChatWithTools(ctx context.Context, systemPrompt string, history []Message, model string) (string, error) {
	// モデル名を設定（config優先、フォールバックはgpt-4o）
	model = GetDefaultModel(model, "openai", "gpt-4o")

	// モデルに応じて API を自動選択
	cfg := config.GetGlobalConfig()
	if cfg.IsResponsesAPIModel(model) {
		return p.chatWithResponses(ctx, systemPrompt, history, model)
	}
	return p.chatWithCompletions(ctx, systemPrompt, history, model)
}

// ChatWithImage は画像付きメッセージで会話を行う
func (p *OpenAIProvider) ChatWithImage(ctx context.Context, systemPrompt string, history []Message, userMessage string, image *ImageData, model string) (string, error) {
	// 画像がない場合は通常のChatWithToolsを使用
	if image == nil || image.Base64 == "" {
		history = append(history, Message{Role: "user", Content: userMessage})
		return p.ChatWithTools(ctx, systemPrompt, history, model)
	}

	// モデル名を設定（config優先、フォールバックはgpt-4o）
	model = GetDefaultModel(model, "openai", "gpt-4o")

	// Responses API モデルの場合は専用の画像処理
	cfg := config.GetGlobalConfig()
	if cfg.IsResponsesAPIModel(model) {
		return p.chatWithImageResponses(ctx, systemPrompt, history, userMessage, image, model)
	}

	return p.chatWithImageCompletions(ctx, systemPrompt, history, userMessage, image, model)
}
