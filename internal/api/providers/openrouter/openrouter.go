package openrouter

import (
	"context"
	"fmt"
	"os"

	"github.com/fatih/color"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func init() {
	api.RegisterProvider("openrouter", func(apiKey string) (api.Provider, error) {
		if apiKey == "" {
			return nil, fmt.Errorf("OPENROUTER_API_KEY not set")
		}
		return New(apiKey), nil
	})
}

var yellow = color.New(color.FgYellow)

const defaultOpenRouterURL = "https://openrouter.ai/api/v1/chat/completions"

// ContentPart はマルチモーダルコンテンツのパート（OpenAI互換）
type ContentPart struct {
	Type     string    `json:"type"`                // "text" or "image_url"
	Text     string    `json:"text,omitempty"`      // type="text"の場合
	ImageURL *ImageURL `json:"image_url,omitempty"` // type="image_url"の場合
}

// ImageURL は画像URL
type ImageURL struct {
	URL string `json:"url"` // "data:image/png;base64,..." 形式
}

// MultimodalMessage はマルチモーダルメッセージ（OpenAI互換）
type MultimodalMessage struct {
	Role    string        `json:"role"`
	Content []ContentPart `json:"content"`
}

// Provider はOpenRouter APIのプロバイダー実装（OpenAI互換 + Claude Compaction対応）
type Provider struct {
	api.BaseProvider
	mcpTools      []api.ToolDefinition // MCP ツール定義（Function Calling用）
	usageCallback api.UsageCallback    // トークン使用量コールバック
	runtimeConfig *config.Config
	toolChoice    *string // tool_choice 強制用
}

// New は新しいProviderを作成
func New(apiKey string) *Provider {
	return &Provider{
		BaseProvider: api.NewBaseProvider("OpenRouter", apiKey, defaultOpenRouterURL, "OPENROUTER_API_URL"),
	}
}

func (p *Provider) effectiveConfig() *config.Config {
	if p != nil && p.runtimeConfig != nil {
		return p.runtimeConfig
	}
	return config.DefaultConfig()
}

func (p *Provider) supportsClaudeCompactionWithConfig(cfg *config.Config, model string) bool {
	if cfg == nil || !cfg.Compression.ClaudeCompaction {
		return false
	}
	if model == "" {
		model = cfg.GetEffectiveModelForProvider("openrouter")
	}
	if model == "" {
		model = "anthropic/claude-sonnet-4.6"
	}
	return isClaudeModel(model) && isCompactionSupported(model)
}

// SupportsClaudeCompaction はこのプロバイダーが Claude Compaction に対応しているかを返す
func (p *Provider) SupportsClaudeCompaction() bool {
	return p.supportsClaudeCompactionWithConfig(p.effectiveConfig(), "")
}

// SupportsClaudeCompactionWithContext は request context とモデルを使って Claude Compaction 対応可否を返す。
func (p *Provider) SupportsClaudeCompactionWithContext(ctx context.Context, model string) bool {
	cfg := p.effectiveConfig()
	if ctxCfg, ok := config.LookupContext(ctx); ok {
		cfg = ctxCfg
	}
	return p.supportsClaudeCompactionWithConfig(cfg, model)
}

// ActiveContextTransport は実際の OpenRouter route に対応する active context transport を返す。
func (p *Provider) ActiveContextTransport(ctx context.Context, model string) api.ActiveContextTransport {
	model = api.GetDefaultModelWithContext(ctx, model, "openrouter", "anthropic/claude-sonnet-4.6")
	cfg := config.ResolveContext(ctx, p.effectiveConfig())
	if p.routePlanForRequest(cfg, model).usesAnthropicMessages() {
		return api.ActiveContextTransportSystemPromptSuffix
	}
	return api.ActiveContextTransportEphemeralSystem
}

// SetRuntimeConfig は provider が参照する runtime 設定を差し替える。
func (p *Provider) SetRuntimeConfig(cfg *config.Config) {
	p.runtimeConfig = cfg
}

// SupportsImages は画像入力対応を返す
func (p *Provider) SupportsImages() bool {
	return true
}

// IsFunctionCallingEnabled は Function Calling が有効かを返す
func (p *Provider) IsFunctionCallingEnabled() bool {
	return os.Getenv("OPENROUTER_FUNCTION_CALLING") != "0"
}

// ChatWithTools は Provider interface の実装
func (p *Provider) ChatWithTools(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	model = api.GetDefaultModelWithContext(ctx, model, "openrouter", "anthropic/claude-sonnet-4.6")
	cfg := config.ResolveContext(ctx, p.effectiveConfig())
	route := p.routePlanForRequest(cfg, model)

	// Claude モデルで context_management が有効なら Anthropic Skin エンドポイントを使用
	if route.usesAnthropicMessages() {
		return p.chatWithClaudeAPI(ctx, systemPrompt, history, "", model, nil, route)
	}

	if api.IsThinkingEnabled(ctx) {
		yellow.Fprintln(api.OutputWriterFromContext(ctx), "⚠️  Warning: OpenRouter does not support Extended Thinking. Proceeding without it.")
	}

	payload, err := p.buildOpenAITextChatPayload(ctx, systemPrompt, history, model)
	if err != nil {
		return "", err
	}
	return p.executeOpenAICompatRequest(ctx, payload, false)
}

// ChatWithImage は画像付きチャットの実装
func (p *Provider) ChatWithImage(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
	if image == nil || image.Base64 == "" {
		history = append(history, api.Message{Role: "user", Content: userMessage})
		return p.ChatWithTools(ctx, systemPrompt, history, model)
	}

	model = api.GetDefaultModelWithContext(ctx, model, "openrouter", "anthropic/claude-sonnet-4.6")
	cfg := config.ResolveContext(ctx, p.effectiveConfig())
	route := p.routePlanForRequest(cfg, model)

	if route.usesAnthropicMessages() {
		return p.chatWithClaudeAPI(ctx, systemPrompt, history, userMessage, model, image, route)
	}

	return p.chatWithImageRequest(ctx, systemPrompt, history, userMessage, image, model)
}

// chatWithImageRequest はOpenAI互換形式での画像送信
func (p *Provider) chatWithImageRequest(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
	payload, err := p.buildOpenAIImageChatPayload(ctx, systemPrompt, history, userMessage, image, model)
	if err != nil {
		return "", err
	}
	return p.executeOpenAICompatRequest(ctx, payload, true)
}

// SetMCPTools はツール定義を設定
func (p *Provider) SetMCPTools(tools []api.ToolDefinition) {
	p.mcpTools = tools
}

// SetUsageCallback はトークン使用量コールバックを設定
func (p *Provider) SetUsageCallback(callback api.UsageCallback) {
	p.usageCallback = callback
}

// SetToolChoice は tool_choice を設定する
func (p *Provider) SetToolChoice(name string) {
	p.toolChoice = &name
}

// ClearToolChoice は tool_choice をクリアする
func (p *Provider) ClearToolChoice() {
	p.toolChoice = nil
}
