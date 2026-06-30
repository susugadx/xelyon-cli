package kimi

import (
	"context"
	"fmt"
	"os"

	"github.com/susugadx/xelyon-cli/internal/api"
	openaicompat "github.com/susugadx/xelyon-cli/internal/api/providers/openai_compat"
	"github.com/susugadx/xelyon-cli/internal/config"
)

func init() {
	api.RegisterProvider("kimi", func(apiKey string) (api.Provider, error) {
		if apiKey == "" {
			return nil, fmt.Errorf("%s not set", kimiAPIKeyEnv)
		}
		return newProvider(apiKey, "kimi"), nil
	})
	api.RegisterProvider("moonshot", func(apiKey string) (api.Provider, error) {
		if apiKey == "" {
			return nil, fmt.Errorf("%s not set", kimiAPIKeyEnv)
		}
		return newProvider(apiKey, "moonshot"), nil
	})
	registerWebSearch("kimi")
	registerWebSearch("moonshot")
}

const (
	kimiAPIKeyEnv                   = "MOONSHOT_API_KEY"
	kimiAPIURLEnv                   = "KIMI_API_URL"
	kimiFunctionCallingEnv          = "KIMI_FUNCTION_CALLING"
	kimiChatCompletionsEndpointPath = "/v1/chat/completions"
	defaultKimiURL                  = "https://api.moonshot.ai" + kimiChatCompletionsEndpointPath
	defaultKimiModel                = "kimi-k2.6"
)

// Provider は Kimi API のプロバイダー実装。
type Provider struct {
	api.BaseProvider
	mcpTools             []api.ToolDefinition
	usageCallback        api.UsageCallback
	lastReasoningContent string
	toolChoice           *string
	functionCalling      *bool
	configKey            string
}

// New は新しい Provider を作成する。
func New(apiKey string) *Provider {
	return newProvider(apiKey, "kimi")
}

func newProvider(apiKey, configKey string) *Provider {
	return &Provider{
		BaseProvider: api.NewBaseProvider("Kimi", apiKey, defaultKimiURL, kimiAPIURLEnv),
		configKey:    config.NormalizeProviderName(configKey),
	}
}

func (p *Provider) configLookupKey() string {
	if p != nil && p.configKey != "" {
		return p.configKey
	}
	return "kimi"
}

// ProviderConfigKey は provider_models の owner key を返す。
func (p *Provider) ProviderConfigKey() string {
	return p.configLookupKey()
}

// SetProviderConfigKey は provider_models の owner key を更新する。
func (p *Provider) SetProviderConfigKey(key string) {
	if p == nil {
		return
	}
	p.configKey = config.NormalizeProviderName(key)
}

// APIURL は API の URL を返す。
func (p *Provider) APIURL() string {
	return p.BaseProvider.APIURL
}

// SupportsImages は画像入力対応を返す。
func (p *Provider) SupportsImages() bool {
	return true
}

// IsFunctionCallingEnabled は Function Calling が有効かを返す。
func (p *Provider) IsFunctionCallingEnabled() bool {
	if p != nil && p.functionCalling != nil {
		return *p.functionCalling
	}
	return os.Getenv(kimiFunctionCallingEnv) != "0"
}

// ChatWithTools は Provider interface の実装。
func (p *Provider) ChatWithTools(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	if err := validateKimiHistoryImages(history); err != nil {
		return "", err
	}
	built := p.buildChatCompletionsRequest(ctx, systemPrompt, history, model)
	req, err := openaicompat.NewBearerJSONRequest(ctx, p.BaseProvider.APIURL, p.APIKey, built.Request)
	if err != nil {
		return "", err
	}

	return openaicompat.RunChatCompletions(ctx, p, req, openaicompat.ChatCompletionsRunOptions{
		SpinnerSuffix:      built.SpinnerSuffix,
		ForceStreaming:     true,
		RequestErrorPrefix: "Kimi API request failed",
		StreamHandler:      p.handleStreamingResponse,
	})
}

// ChatWithImage は Kimi Chat Completions の multimodal message として画像付きメッセージを送信する。
func (p *Provider) ChatWithImage(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
	history = append(history, api.NewUserMessageWithOptionalImage(userMessage, image))
	return p.ChatWithTools(ctx, systemPrompt, history, model)
}

// SetMCPTools は MCP ツール定義を設定する。
func (p *Provider) SetMCPTools(tools []api.ToolDefinition) {
	p.mcpTools = tools
}

// SetUsageCallback は使用量レポートのコールバックを設定する。
func (p *Provider) SetUsageCallback(callback api.UsageCallback) {
	p.usageCallback = callback
}

// SetToolChoice は tool_choice を設定する。
func (p *Provider) SetToolChoice(name string) {
	p.toolChoice = &name
}

// ClearToolChoice は tool_choice をクリアする。
func (p *Provider) ClearToolChoice() {
	p.toolChoice = nil
}

// LastReasoningContent は最後の API 呼び出しで返された reasoning_content を返す。
func (p *Provider) LastReasoningContent() string {
	return p.lastReasoningContent
}
