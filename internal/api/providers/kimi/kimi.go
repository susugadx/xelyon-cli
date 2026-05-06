package kimi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/api/providers/openai"
	openaicompat "github.com/susugadx/xelyon-cli/internal/api/providers/openai_compat"
	openaicompatstream "github.com/susugadx/xelyon-cli/internal/api/providers/openai_compat_stream"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
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
}

var yellow = color.New(color.FgYellow)

const (
	kimiAPIKeyEnv    = "MOONSHOT_API_KEY"
	kimiAPIURLEnv    = "KIMI_API_URL"
	defaultKimiURL   = "https://api.moonshot.ai/v1/chat/completions"
	defaultKimiModel = "kimi-k2.6"
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
	return false
}

// IsFunctionCallingEnabled は Function Calling が有効かを返す。
func (p *Provider) IsFunctionCallingEnabled() bool {
	if p != nil && p.functionCalling != nil {
		return *p.functionCalling
	}
	return os.Getenv("KIMI_FUNCTION_CALLING") != "0"
}

// ChatWithTools は Provider interface の実装。
func (p *Provider) ChatWithTools(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
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

func kimiThinkingConfig(ctx context.Context, providerConfigKey, requestedModel string) (map[string]any, bool, string) {
	catalogModel := kimiCatalogModel(ctx, providerConfigKey, requestedModel)
	if isKimiForcedThinkingModel(requestedModel) || isKimiForcedThinkingModel(catalogModel) {
		if api.IsThinkingEnabled(ctx) {
			return kimiThinkingEnabledField(kimiThinkingPayloadModel(requestedModel, catalogModel)), true, "Reasoner"
		}
		return nil, true, "Reasoner"
	}
	if !isKimiConfigurableThinkingModel(requestedModel) && !isKimiConfigurableThinkingModel(catalogModel) {
		return nil, false, ""
	}
	if api.IsThinkingEnabled(ctx) {
		return kimiThinkingEnabledField(kimiThinkingPayloadModel(requestedModel, catalogModel)), true, "Reasoner"
	}
	return map[string]any{
		"thinking": map[string]any{"type": "disabled"},
	}, false, ""
}

func kimiThinkingEnabledField(model string) map[string]any {
	thinking := map[string]any{"type": "enabled"}
	if isKimiKeepAllThinkingModel(model) {
		thinking["keep"] = "all"
	}
	return map[string]any{
		"thinking": thinking,
	}
}

func kimiCatalogModel(ctx context.Context, providerConfigKey, requestedModel string) string {
	catalogModel := config.FromContext(ctx).ModelCatalogName(providerConfigKey, requestedModel)
	if strings.TrimSpace(catalogModel) == "" {
		return requestedModel
	}
	return catalogModel
}

func kimiThinkingPayloadModel(requestedModel, catalogModel string) string {
	if isKimiConfigurableThinkingModel(catalogModel) || isKimiForcedThinkingModel(catalogModel) {
		return catalogModel
	}
	return requestedModel
}

func isKimiConfigurableThinkingModel(model string) bool {
	switch strings.ToLower(strings.TrimSpace(model)) {
	case "kimi-k2.6", "kimi-k2.5":
		return true
	default:
		return false
	}
}

func isKimiForcedThinkingModel(model string) bool {
	return strings.EqualFold(strings.TrimSpace(model), "kimi-k2-thinking")
}

func isKimiKeepAllThinkingModel(model string) bool {
	return strings.EqualFold(strings.TrimSpace(model), "kimi-k2.6")
}

// handleStreamingResponse はストリーミングレスポンスを処理する。
func (p *Provider) handleStreamingResponse(ctx context.Context, resp *http.Response, spinner *ui.Spinner) (string, error) {
	out := api.OutputWriterFromContext(ctx)
	dim := color.New(color.Faint)
	reasoningActive := false
	p.lastReasoningContent = ""

	streamResult, err := openaicompatstream.ParseSSEStream(ctx, resp, spinner, openaicompatstream.ParseSSEOptions{
		ValidateData: func(data string) error {
			if err := api.ValidateStreamResponse([]byte(data)); err != nil {
				return fmt.Errorf("invalid response structure: %w", err)
			}
			return nil
		},
		UsageDecoder: decodeKimiUsage,
		OnReasoningContent: func(content string, first bool) {
			if first {
				reasoningActive = true
				spinner.Stop()
				dim.Fprint(out, "💭 ")
			}
			dim.Fprint(out, content)
		},
		OnReasoningBoundary: func() {
			if !reasoningActive {
				return
			}
			_, _ = fmt.Fprintln(out)
			_, _ = fmt.Fprintln(out)
			reasoningActive = false
		},
		OnToolCallArguments: func(toolName string) {
			if !spinner.IsActive() {
				spinner.Start(ui.SpinnerMessageForTool(toolName))
			}
		},
		StopOnToolCallsFinish: true,
	})
	if err != nil {
		return "", err
	}

	p.lastReasoningContent = streamResult.ReasoningContent
	if streamResult.Usage != nil && p.usageCallback != nil {
		p.usageCallback(*streamResult.Usage)
	}

	return openaicompatstream.BuildContentWithToolCalls(
		streamResult.Content,
		streamResult.ToolCalls,
		openai.ConvertToolCallToToolJSON,
	), nil
}

func decodeKimiUsage(raw json.RawMessage) (*api.Usage, error) {
	if !openaicompatstream.HasUsagePayload(raw) {
		return nil, nil
	}

	var usagePayload struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		CachedTokens     int `json:"cached_tokens,omitempty"`
	}
	if err := json.Unmarshal(raw, &usagePayload); err != nil {
		return nil, err
	}

	return &api.Usage{
		InputTokens:       usagePayload.PromptTokens,
		OutputTokens:      usagePayload.CompletionTokens,
		CachedInputTokens: usagePayload.CachedTokens,
	}, nil
}

// ChatWithImage は画像付きメッセージで会話を行う（非対応: テキストのみ送信）。
func (p *Provider) ChatWithImage(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
	if image != nil && image.Base64 != "" {
		yellow.Fprintln(api.OutputWriterFromContext(ctx), "Warning: Kimi does not support image input. The image will be ignored.")
	}
	history = append(history, api.Message{Role: "user", Content: userMessage})
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
