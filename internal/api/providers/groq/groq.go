package groq

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/fatih/color"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/api/providers/openai"
	openaicompat "github.com/susugadx/xelyon-cli/internal/api/providers/openai_compat"
	openaicompatstream "github.com/susugadx/xelyon-cli/internal/api/providers/openai_compat_stream"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func init() {
	api.RegisterProvider("groq", func(apiKey string) (api.Provider, error) {
		if apiKey == "" {
			return nil, fmt.Errorf("GROQ_API_KEY not set")
		}
		return New(apiKey), nil
	})
}

var yellow = color.New(color.FgYellow)

const (
	groqChatCompletionsEndpointPath = "/openai/v1/chat/completions"
	defaultGroqURL                  = "https://api.groq.com" + groqChatCompletionsEndpointPath
	groqFunctionCallingEnv          = "GROQ_FUNCTION_CALLING"
)

// Provider はGroq APIのプロバイダー実装（OpenAI互換）
type Provider struct {
	api.BaseProvider
	mcpTools      []api.ToolDefinition // MCP ツール定義（Function Calling用）
	usageCallback api.UsageCallback    // トークン使用量コールバック
	toolChoice    *string              // tool_choice 強制用
}

// New は新しいProviderを作成
func New(apiKey string) *Provider {
	return &Provider{
		BaseProvider: api.NewBaseProvider("Groq", apiKey, defaultGroqURL, "GROQ_API_URL"),
	}
}

// SupportsImages は画像入力対応を返す
func (p *Provider) SupportsImages() bool {
	return false
}

// IsFunctionCallingEnabled は Function Calling が有効かを返す
// GROQ_FUNCTION_CALLING=0 で無効化可能
func (p *Provider) IsFunctionCallingEnabled() bool {
	return os.Getenv(groqFunctionCallingEnv) != "0"
}

// ChatWithTools は Provider interface の実装（context対応）
func (p *Provider) ChatWithTools(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	// Extended Thinking 非対応警告
	if api.IsThinkingEnabled(ctx) {
		yellow.Fprintln(api.OutputWriterFromContext(ctx), "⚠️  Warning: Groq does not support Extended Thinking. Proceeding without it.")
	}

	reqBody := p.buildChatCompletionsRequest(ctx, systemPrompt, history, model)
	req, err := openaicompat.NewBearerJSONRequest(ctx, p.BaseProvider.APIURL, p.APIKey, reqBody)
	if err != nil {
		return "", err
	}

	return openaicompat.RunChatCompletions(ctx, p, req, openaicompat.ChatCompletionsRunOptions{
		StreamHandler:    p.handleStreamingResponse,
		NonStreamHandler: p.handleNonStreamingResponse,
	})
}

func (p *Provider) buildChatCompletionsRequest(ctx context.Context, systemPrompt string, history []api.Message, model string) openaicompat.ChatCompletionsRequest {
	// モデル名を設定（config優先、フォールバックはkimi-k2-instruct）
	model = api.GetDefaultModelWithContext(ctx, model, "groq", "moonshotai/kimi-k2-instruct")
	options := openaicompat.ChatCompletionsRequestOptions{
		Model:         model,
		SystemPrompt:  systemPrompt,
		ActiveContext: api.ActiveContextBlocksFromContext(ctx),
		History:       history,
		MaxTokens:     api.GetMaxOutputTokens(ctx, "groq", model),
		Stream:        true,
		IncludeUsage:  true,
	}

	// Function Calling: ツール定義を追加（環境変数で無効化可能）
	if api.ShouldSendToolPayload(ctx, p.IsFunctionCallingEnabled()) {
		options.FunctionCalling = &openaicompat.FunctionCallingOptions{
			Tools:    openai.GetCombinedOpenAIToolsWithContext(ctx, p.mcpTools),
			ToolName: p.toolChoice,
		}
	}

	return openaicompat.BuildChatCompletionsRequest(options)
}

// handleStreamingResponse はストリーミングレスポンスを処理
func (p *Provider) handleStreamingResponse(ctx context.Context, resp *http.Response, spinner *ui.Spinner) (string, error) {
	errOut := api.ErrorWriterFromContext(ctx)
	streamResult, err := openaicompatstream.ParseSSEStream(ctx, resp, spinner, openaicompatstream.ParseSSEOptions{
		OnChunkDecodeError: func(parseErr error) error {
			// JSONパースエラーは警告して継続（既存方針を維持）
			fmt.Fprintf(errOut, "⚠️  Warning: failed to parse streaming response: %v\n", parseErr)
			return nil
		},
		OnUsageDecodeError: func(parseErr error) error {
			fmt.Fprintf(errOut, "⚠️  Warning: failed to parse streaming response: %v\n", parseErr)
			return nil
		},
		OnToolCallArguments: func(toolName string) {
			if !spinner.IsActive() {
				spinner.Start(ui.SpinnerMessageForTool(toolName))
			}
		},
	})
	if err != nil {
		return "", err
	}

	// Usage callback
	if p.usageCallback != nil && streamResult.Usage != nil {
		p.usageCallback(*streamResult.Usage)
	}

	return openaicompatstream.BuildContentWithToolCalls(
		streamResult.Content,
		streamResult.ToolCalls,
		openai.ConvertToolCallToToolJSON,
	), nil
}

// handleNonStreamingResponse は非ストリーミングレスポンスを処理（フォールバック）
func (p *Provider) handleNonStreamingResponse(ctx context.Context, resp *http.Response, spinner *ui.Spinner) (string, error) {
	return api.HandleNonStreamingResponse(ctx, resp, spinner)
}

// ChatWithImage は画像付きメッセージで会話を行う（非対応：テキストのみ送信）
func (p *Provider) ChatWithImage(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
	// Groqは画像非対応なので警告を出してテキストのみ送信
	if image != nil && image.Base64 != "" {
		yellow.Fprintln(api.OutputWriterFromContext(ctx), "Warning: Groq does not support image input. The image will be ignored.")
	}
	history = append(history, api.Message{Role: "user", Content: userMessage})
	return p.ChatWithTools(ctx, systemPrompt, history, model)
}

// APIURL はテスト用にAPIURLを公開
func (p *Provider) APIURL() string {
	return p.BaseProvider.APIURL
}

// SetMCPTools は MCP ツール定義を設定する（Function Calling用）
func (p *Provider) SetMCPTools(tools []api.ToolDefinition) {
	p.mcpTools = tools
}

// SetUsageCallback は使用量レポートのコールバックを設定する
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
