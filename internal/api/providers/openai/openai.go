package openai

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func init() {
	api.RegisterProvider("openai", func(apiKey string) (api.Provider, error) {
		if apiKey == "" {
			return nil, fmt.Errorf("OPENAI_API_KEY not set")
		}
		return New(apiKey), nil
	})
}

const defaultOpenAIURL = "https://api.openai.com/v1/chat/completions"

// Provider はOpenAI APIのプロバイダー実装
type Provider struct {
	apiKey         string
	apiURL         string
	httpClient     *http.Client
	lastResponseID string                   // Responses API の最新レスポンスID（キャッシュ用）
	mcpTools       []api.OpenAIToolFunction // MCP ツール定義（Function Calling用）
	usageCallback  api.UsageCallback        // トークン使用量コールバック
}

// New は新しいProviderを作成
func New(apiKey string) *Provider {
	// 環境変数からURLをオーバーライド可能
	apiURL := os.Getenv("OPENAI_API_URL")
	if apiURL == "" {
		apiURL = defaultOpenAIURL
	}

	return &Provider{
		apiKey: apiKey,
		apiURL: apiURL,
		httpClient: &http.Client{
			Timeout: config.DefaultHTTPTimeout,
		},
	}
}

// Name はプロバイダー名を返す
func (p *Provider) Name() string {
	return "OpenAI"
}

// SupportsImages は画像入力対応を返す
func (p *Provider) SupportsImages() bool {
	return true
}

// IsFunctionCallingEnabled は Function Calling が有効かを返す
func (p *Provider) IsFunctionCallingEnabled() bool {
	return true
}

// LevelToReasoningEffort は Thinking Level を OpenAI reasoning_effort に変換
func LevelToReasoningEffort(level string) string {
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
func (p *Provider) ChatWithTools(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	// モデル名を設定（config優先、フォールバックはgpt-4o）
	model = api.GetDefaultModel(model, "openai", "gpt-4o")

	// モデルに応じて API を自動選択
	cfg := config.GetGlobalConfig()
	isResponses := cfg.IsResponsesAPIModel(model)
	if os.Getenv("XELYON_DEBUG_OPENAI") == "1" {
		fmt.Printf("[DEBUG OpenAI] ChatWithTools model=%s, isResponsesAPI=%v\n", model, isResponses)
	}
	if isResponses {
		return p.chatWithResponses(ctx, systemPrompt, history, model)
	}
	return p.chatWithCompletions(ctx, systemPrompt, history, model)
}

// HasCachedResponseID は Responses API のキャッシュ済み responseID があるか返す
func (p *Provider) HasCachedResponseID() bool {
	return p.lastResponseID != ""
}

// ClearResponseID は responseID をクリアする（圧縮後などに使用）
func (p *Provider) ClearResponseID() {
	p.lastResponseID = ""
}

// SetMCPTools は MCP ツール定義を設定する（Function Calling用）
func (p *Provider) SetMCPTools(tools []api.OpenAIToolFunction) {
	p.mcpTools = tools
}

// SetUsageCallback は使用量レポートのコールバックを設定する
func (p *Provider) SetUsageCallback(callback api.UsageCallback) {
	p.usageCallback = callback
}

// ChatWithImage は画像付きメッセージで会話を行う
func (p *Provider) ChatWithImage(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
	// 画像がない場合は通常のChatWithToolsを使用
	if image == nil || image.Base64 == "" {
		history = append(history, api.Message{Role: "user", Content: userMessage})
		return p.ChatWithTools(ctx, systemPrompt, history, model)
	}

	// モデル名を設定（config優先、フォールバックはgpt-4o）
	model = api.GetDefaultModel(model, "openai", "gpt-4o")

	// Responses API モデルの場合は専用の画像処理
	cfg := config.GetGlobalConfig()
	if cfg.IsResponsesAPIModel(model) {
		return p.chatWithImageResponses(ctx, systemPrompt, history, userMessage, image, model)
	}

	return p.chatWithImageCompletions(ctx, systemPrompt, history, userMessage, image, model)
}
