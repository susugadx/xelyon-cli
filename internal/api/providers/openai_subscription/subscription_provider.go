package openaisubscription

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	openairesponses "github.com/susugadx/xelyon-cli/internal/api/providers/openai_responses"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

const subscriptionLoginCommand = "xelyon auth openai-subscription login"

// ResponsesRequest は subscription Responses-shaped request を表します。
type ResponsesRequest = openairesponses.Request

func init() {
	api.RegisterProvider(subscriptionProviderKey, func(_ string) (api.Provider, error) {
		return NewSubscription(), nil
	})
}

// New は openai_subscription provider を作成します。
func New() api.Provider {
	return NewSubscription()
}

// SubscriptionProvider は ChatGPT/Codex OAuth subscription backend 用の experimental provider です。
type SubscriptionProvider struct {
	api.BaseProvider
	mcpTools                 []api.ToolDefinition
	usageCallback            api.UsageCallback
	toolChoice               *string
	responsesRequestObserver func(ResponsesRequest)
	configKey                string
	lastResponsesInputItems  []api.InputItem
}

// NewSubscription は openai_subscription provider を作成します。
func NewSubscription() *SubscriptionProvider {
	return &SubscriptionProvider{
		BaseProvider: api.NewBaseProvider(subscriptionDisplayName, "", subscriptionDefaultEndpoint(), ""),
		configKey:    subscriptionProviderKey,
	}
}

func subscriptionDefaultEndpoint() string {
	return subscriptionDefaultEndpointURL
}

// RuntimeProviderName は実行時 provider key を返します。
func (p *SubscriptionProvider) RuntimeProviderName() string {
	return subscriptionProviderKey
}

// ProviderConfigKey は provider_models/session owner key を返します。
func (p *SubscriptionProvider) ProviderConfigKey() string {
	if p != nil && strings.TrimSpace(p.configKey) != "" {
		return subscriptionProviderKey
	}
	return subscriptionProviderKey
}

// SetProviderConfigKey は provider_models/session owner key を設定します。
func (p *SubscriptionProvider) SetProviderConfigKey(_ string) {
	if p == nil {
		return
	}
	p.configKey = subscriptionProviderKey
}

// SupportsImages は subscription provider の画像入力対応を返します。
func (p *SubscriptionProvider) SupportsImages() bool {
	return false
}

// IsFunctionCallingEnabled は subscription provider の Function Calling 対応を返します。
func (p *SubscriptionProvider) IsFunctionCallingEnabled() bool {
	return true
}

// SetMCPTools は MCP ツール定義を設定します。
func (p *SubscriptionProvider) SetMCPTools(tools []api.ToolDefinition) {
	p.mcpTools = tools
}

// SetUsageCallback は使用量 callback を設定します。
func (p *SubscriptionProvider) SetUsageCallback(callback api.UsageCallback) {
	p.usageCallback = callback
}

// SetToolChoice は subscription Responses request の tool_choice を設定します。
func (p *SubscriptionProvider) SetToolChoice(name string) {
	if p == nil {
		return
	}
	p.toolChoice = &name
}

// ClearToolChoice は subscription Responses request の tool_choice を解除します。
func (p *SubscriptionProvider) ClearToolChoice() {
	if p == nil {
		return
	}
	p.toolChoice = nil
}

// LastOpenAIResponsesInputItems は最後の subscription Responses 応答から得た replay items を返します。
func (p *SubscriptionProvider) LastOpenAIResponsesInputItems() []api.InputItem {
	if p == nil {
		return nil
	}
	return api.CloneInputItems(p.lastResponsesInputItems)
}

func (p *SubscriptionProvider) setLastOpenAIResponsesInputItems(items []api.InputItem) {
	if p == nil {
		return
	}
	p.lastResponsesInputItems = api.CloneInputItems(items)
}

func (p *SubscriptionProvider) clearLastOpenAIResponsesInputItems() {
	if p == nil {
		return
	}
	p.lastResponsesInputItems = nil
}

// ChatWithTools は subscription backend へ送る会話 request を実行します。
func (p *SubscriptionProvider) ChatWithTools(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	model = api.ResolveProviderRequestModel(ctx, model, subscriptionProviderKey)
	if err := ValidateSubscriptionModel(model); err != nil {
		return "", err
	}
	return p.chatWithResponses(ctx, systemPrompt, history, model)
}

// ChatWithImage は画像入力がない場合だけ通常 request と同じ path を使います。
func (p *SubscriptionProvider) ChatWithImage(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
	if image == nil || image.Base64 == "" {
		history = append(history, api.Message{Role: "user", Content: userMessage})
		return p.ChatWithTools(ctx, systemPrompt, history, model)
	}
	return "", fmt.Errorf("openai_subscription does not support image input")
}

func (p *SubscriptionProvider) chatWithResponses(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	p.clearLastOpenAIResponsesInputItems()
	endpoint, err := validateSubscriptionResponsesEndpoint(DefaultSubscriptionAuthConfig().Endpoint)
	if err != nil {
		return "", err
	}
	content, _, err := openairesponses.RunResponsesRequest(ctx, openairesponses.RunOptions{
		URL: endpoint,
		BuildRequest: func() ResponsesRequest {
			return p.buildChatResponsesRequest(ctx, systemPrompt, history, model)
		},
		PrepareRequest: p.prepareSubscriptionResponsesRequest,
		ExecuteRequest: p.executeSubscriptionResponsesRequest,
		HandleStreaming: func(ctx context.Context, resp *http.Response, spinner *ui.Spinner) (string, string, error) {
			return p.handleSubscriptionResponsesStreaming(ctx, resp, spinner)
		},
		HandleNonStreaming: func(ctx context.Context, resp *http.Response, spinner *ui.Spinner) (string, string, error) {
			return p.handleSubscriptionResponsesNonStreaming(ctx, resp, spinner)
		},
		HandleHTTPError:          handleSubscriptionHTTPError,
		RequestObserver:          p.responsesRequestObserver,
		SetLocalAutoCompressSkip: func(bool) {},
		ProviderName:             subscriptionDisplayName,
		DebugName:                subscriptionDisplayName,
		Debug:                    subscriptionDebugEnabled(),
		DebugWriter:              api.ErrorWriterFromContext(ctx),
		DebugRequestPreview:      subscriptionDebugRequestPreview,
	})
	return content, err
}

func (p *SubscriptionProvider) buildChatResponsesRequest(ctx context.Context, systemPrompt string, history []api.Message, model string) ResponsesRequest {
	modelIdentity := subscriptionModelIdentity(ctx, model)
	activeContext := openairesponses.ActiveContextFromContext(ctx)
	return openairesponses.BuildChatRequest(openairesponses.ChatRequestOptions{
		Base: openairesponses.BaseRequestOptions{
			Model:                modelIdentity,
			Stream:               true,
			Store:                false,
			PromptCacheKey:       openairesponses.BuildPromptCacheKey(modelIdentity.RequestName(), systemPrompt),
			Instructions:         systemPrompt,
			Tools:                subscriptionResponsesTools(ctx, p.mcpTools, p.toolChoice, p.IsFunctionCallingEnabled()),
			ToolChoice:           subscriptionResponsesToolChoice(ctx, p.toolChoice, p.IsFunctionCallingEnabled()),
			Reasoning:            subscriptionResponsesReasoningConfig(ctx, modelIdentity),
			PromptCacheRetention: "",
		},
		RequestContext: ctx,
		SystemPrompt:   systemPrompt,
		CompactedInput: api.CompactedInputItemsFromContext(ctx),
		ActiveContext:  activeContext,
		History:        history,
	})
}

func (p *SubscriptionProvider) executeSubscriptionResponsesRequest(req *http.Request, stream bool) (*http.Response, error) {
	if stream {
		return p.ExecuteRequest(req)
	}
	return p.executeSubscriptionLongRunningRequest(req)
}

func (p *SubscriptionProvider) executeSubscriptionLongRunningRequest(req *http.Request) (*http.Response, error) {
	return api.DoWithRetry(req.Context(), openairesponses.NewLongRunningHTTPClient(p.HTTPClient), req)
}

func (p *SubscriptionProvider) handleSubscriptionResponsesStreaming(ctx context.Context, resp *http.Response, spinner *ui.Spinner) (string, string, error) {
	debugEnabled := subscriptionDebugEnabled()
	debugRawPayload := false
	return openairesponses.HandleStreaming(ctx, resp, spinner, openairesponses.StreamingOptions{
		ProviderName:        subscriptionDisplayName,
		DebugName:           subscriptionDisplayName,
		Debug:               debugEnabled,
		DebugOverride:       &debugEnabled,
		DebugRawPayload:     &debugRawPayload,
		DebugWriter:         api.ErrorWriterFromContext(ctx),
		UsageCallback:       p.usageCallback,
		ReplayItemsCallback: p.setLastOpenAIResponsesInputItems,
	})
}

func subscriptionModelIdentity(ctx context.Context, model string) openairesponses.ModelIdentity {
	cfg := config.FromContext(ctx)
	return openairesponses.NewModelIdentity(model, cfg.ModelCatalogName(subscriptionProviderKey, model))
}

func subscriptionResponsesTools(ctx context.Context, mcpTools []api.ToolDefinition, toolChoice *string, functionCallingEnabled bool) []openairesponses.Tool {
	if !api.ShouldSendToolPayload(ctx, functionCallingEnabled) {
		return nil
	}
	return openairesponses.BuildToolDefinitionsWithContext(ctx, mcpTools)
}

func subscriptionResponsesToolChoice(ctx context.Context, toolChoice *string, functionCallingEnabled bool) interface{} {
	if !api.ShouldSendToolPayload(ctx, functionCallingEnabled) {
		return nil
	}
	return openairesponses.BuildFunctionToolChoice(toolChoice)
}

func subscriptionResponsesReasoningConfig(ctx context.Context, model openairesponses.ModelIdentity) *openairesponses.ReasoningConfig {
	cfg := config.FromContext(ctx)
	if api.IsThinkingEnabled(ctx) {
		return &openairesponses.ReasoningConfig{
			Effort: openairesponses.ReasoningEffortFromThinkingLevel(cfg.Thinking.Level),
		}
	}
	if strings.Contains(strings.ToLower(model.CatalogName()), "codex") {
		return &openairesponses.ReasoningConfig{Effort: "low"}
	}
	return nil
}

func subscriptionDebugEnabled() bool {
	return os.Getenv("XELYON_DEBUG_OPENAI_SUBSCRIPTION") == "1"
}

func (p *SubscriptionProvider) handleSubscriptionResponsesNonStreaming(ctx context.Context, resp *http.Response, spinner *ui.Spinner) (string, string, error) {
	return openairesponses.HandleNonStreaming(ctx, resp, spinner, openairesponses.NonStreamingOptions{
		ProviderName:        subscriptionDisplayName,
		UsageCallback:       p.usageCallback,
		ReplayItemsCallback: p.setLastOpenAIResponsesInputItems,
	})
}
