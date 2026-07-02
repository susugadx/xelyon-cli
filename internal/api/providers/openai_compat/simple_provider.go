package openaicompat

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/fatih/color"
	"github.com/susugadx/xelyon-cli/internal/api"
	openaicompatstream "github.com/susugadx/xelyon-cli/internal/api/providers/openai_compat_stream"
	"github.com/susugadx/xelyon-cli/internal/uiruntime"
	"github.com/susugadx/xelyon-cli/internal/uitoolview"
)

// SimpleProviderSpec は単純な OpenAI 互換 Chat Completions provider の差分を表す。
type SimpleProviderSpec struct {
	ProviderKey                       string
	DisplayName                       string
	DefaultURL                        string
	URLOverrideEnv                    string
	FunctionCallingEnv                string
	SupportsImages                    bool
	ThinkingUnsupportedWarning        string
	ImageUnsupportedWarning           string
	WarnAndContinueOnStreamParseError bool
	BuildTools                        func(context.Context, []api.ToolDefinition) []api.OpenAITool
	EncodeToolCall                    func(*api.OpenAIToolCall) (string, error)
}

// SimpleProvider は OpenAI 互換 Chat Completions だけを使う provider の共通実装。
type SimpleProvider struct {
	api.BaseProvider
	spec          SimpleProviderSpec
	mcpTools      []api.ToolDefinition
	usageCallback api.UsageCallback
	toolChoice    *string
}

// NewSimpleProvider は spec に基づいて OpenAI 互換 provider を作成する。
func NewSimpleProvider(apiKey string, spec SimpleProviderSpec) *SimpleProvider {
	return &SimpleProvider{
		BaseProvider: api.NewBaseProvider(spec.DisplayName, apiKey, spec.DefaultURL, spec.URLOverrideEnv),
		spec:         spec,
	}
}

// SupportsImages は画像入力対応を返す。
func (p *SimpleProvider) SupportsImages() bool {
	return p.spec.SupportsImages
}

// IsFunctionCallingEnabled は Function Calling が有効かを返す。
func (p *SimpleProvider) IsFunctionCallingEnabled() bool {
	if p.spec.FunctionCallingEnv == "" {
		return true
	}
	return os.Getenv(p.spec.FunctionCallingEnv) != "0"
}

// ChatWithTools は OpenAI 互換 Chat Completions で会話する。
func (p *SimpleProvider) ChatWithTools(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	if api.IsThinkingEnabled(ctx) && p.spec.ThinkingUnsupportedWarning != "" {
		color.New(color.FgYellow).Fprintln(api.OutputWriterFromContext(ctx), p.spec.ThinkingUnsupportedWarning)
	}

	reqBody := p.buildChatCompletionsRequest(ctx, systemPrompt, history, model)
	req, err := NewBearerJSONRequest(ctx, p.BaseProvider.APIURL, p.APIKey, reqBody)
	if err != nil {
		return "", err
	}

	return RunChatCompletions(ctx, p, req, ChatCompletionsRunOptions{
		StreamHandler:    p.handleStreamingResponse,
		NonStreamHandler: p.handleNonStreamingResponse,
	})
}

func (p *SimpleProvider) buildChatCompletionsRequest(ctx context.Context, systemPrompt string, history []api.Message, model string) ChatCompletionsRequest {
	providerKey := p.spec.ProviderKey
	model = api.ResolveProviderRequestModel(ctx, model, providerKey)
	options := ChatCompletionsRequestOptions{
		Model:            model,
		SystemPrompt:     systemPrompt,
		ActiveContext:    api.ActiveContextBlocksFromContext(ctx),
		History:          history,
		MaxTokens:        api.GetMaxOutputTokens(ctx, providerKey, model),
		Stream:           true,
		IncludeUsage:     true,
		ImagePayloadMode: imagePayloadModeForSupport(p.SupportsImages()),
	}

	if api.ShouldSendToolPayload(ctx, p.IsFunctionCallingEnabled()) {
		options.FunctionCalling = &FunctionCallingOptions{
			Tools:    p.buildTools(ctx),
			ToolName: p.toolChoice,
		}
	}

	return BuildChatCompletionsRequest(options)
}

func imagePayloadModeForSupport(supportsImages bool) ImagePayloadMode {
	if supportsImages {
		return ImagePayloadMultimodal
	}
	return ImagePayloadTextOnly
}

// BuildChatCompletionsRequest は preview / diagnostics 用に送信前 payload を構築する。
func (p *SimpleProvider) BuildChatCompletionsRequest(ctx context.Context, systemPrompt string, history []api.Message, model string) ChatCompletionsRequest {
	return p.buildChatCompletionsRequest(ctx, systemPrompt, history, model)
}

func (p *SimpleProvider) buildTools(ctx context.Context) []api.OpenAITool {
	if p.spec.BuildTools == nil {
		return nil
	}
	return p.spec.BuildTools(ctx, p.mcpTools)
}

func (p *SimpleProvider) handleStreamingResponse(ctx context.Context, resp *http.Response, spinner *uiruntime.Spinner) (string, error) {
	errOut := api.ErrorWriterFromContext(ctx)
	streamResult, err := openaicompatstream.ParseSSEStream(ctx, resp, spinner, openaicompatstream.ParseSSEOptions{
		OnChunkDecodeError: p.streamParseWarning(errOut),
		OnUsageDecodeError: p.streamParseWarning(errOut),
		OnToolCallArguments: func(toolName string) {
			if !spinner.IsActive() {
				spinner.Start(uitoolview.SpinnerMessageForTool(toolName))
			}
		},
	})
	if err != nil {
		return "", err
	}

	if p.usageCallback != nil && streamResult.Usage != nil {
		p.usageCallback(*streamResult.Usage)
	}

	return openaicompatstream.BuildContentWithToolCalls(
		streamResult.Content,
		streamResult.ToolCalls,
		p.spec.EncodeToolCall,
	), nil
}

func (p *SimpleProvider) streamParseWarning(errOut io.Writer) func(error) error {
	if !p.spec.WarnAndContinueOnStreamParseError {
		return nil
	}
	return func(parseErr error) error {
		fmt.Fprintf(errOut, "⚠️  Warning: failed to parse streaming response: %v\n", parseErr)
		return nil
	}
}

func (p *SimpleProvider) handleNonStreamingResponse(ctx context.Context, resp *http.Response, spinner *uiruntime.Spinner) (string, error) {
	return api.HandleNonStreamingResponse(ctx, resp, spinner)
}

// ChatWithImage は画像付きメッセージを処理する。
func (p *SimpleProvider) ChatWithImage(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
	if image != nil && image.Base64 != "" && !p.SupportsImages() && p.spec.ImageUnsupportedWarning != "" {
		color.New(color.FgYellow).Fprintln(api.OutputWriterFromContext(ctx), p.spec.ImageUnsupportedWarning)
	}
	history = append(history, api.NewUserMessageWithOptionalImage(userMessage, image))
	return p.ChatWithTools(ctx, systemPrompt, history, model)
}

// APIURL はテスト用に API URL を返す。
func (p *SimpleProvider) APIURL() string {
	return p.BaseProvider.APIURL
}

// SetMCPTools は MCP ツール定義を設定する。
func (p *SimpleProvider) SetMCPTools(tools []api.ToolDefinition) {
	p.mcpTools = tools
}

// MCPTools は設定済み MCP ツール定義を返す。
func (p *SimpleProvider) MCPTools() []api.ToolDefinition {
	return append([]api.ToolDefinition(nil), p.mcpTools...)
}

// SetUsageCallback は使用量レポートのコールバックを設定する。
func (p *SimpleProvider) SetUsageCallback(callback api.UsageCallback) {
	p.usageCallback = callback
}

// SetToolChoice は tool_choice を設定する。
func (p *SimpleProvider) SetToolChoice(name string) {
	p.toolChoice = &name
}

// ClearToolChoice は tool_choice をクリアする。
func (p *SimpleProvider) ClearToolChoice() {
	p.toolChoice = nil
}

// ToolChoice は現在の tool_choice 強制値を返す。
func (p *SimpleProvider) ToolChoice() *string {
	if p.toolChoice == nil {
		return nil
	}
	value := *p.toolChoice
	return &value
}
