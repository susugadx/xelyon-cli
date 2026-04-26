package deepseek

import (
	"context"
	"encoding/json"
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
	api.RegisterProvider("deepseek", func(apiKey string) (api.Provider, error) {
		if apiKey == "" {
			return nil, fmt.Errorf("DEEPSEEK_API_KEY not set")
		}
		return New(apiKey), nil
	})
}

var yellow = color.New(color.FgYellow)

const defaultDeepSeekURL = "https://api.deepseek.com/chat/completions"

// Provider はDeepSeek APIのプロバイダー実装
type Provider struct {
	api.BaseProvider
	mcpTools             []api.ToolDefinition // MCP ツール定義（Function Calling用）
	usageCallback        api.UsageCallback    // トークン使用量コールバック
	lastReasoningContent string               // 最後の reasoning_content（DeepSeek Reasoner用）
	toolChoice           *string              // tool_choice 強制用
}

// New は新しいProviderを作成
func New(apiKey string) *Provider {
	return &Provider{
		BaseProvider: api.NewBaseProvider("DeepSeek", apiKey, defaultDeepSeekURL, "DEEPSEEK_API_URL"),
	}
}

// Name はプロバイダー名を返す
func (p *Provider) Name() string {
	return "DeepSeek"
}

// APIURL はAPIのURLを返す
func (p *Provider) APIURL() string {
	return p.BaseProvider.APIURL
}

// SupportsImages は画像入力対応を返す
func (p *Provider) SupportsImages() bool {
	return false
}

// IsFunctionCallingEnabled は Function Calling が有効かを返す
// DEEPSEEK_FUNCTION_CALLING=0 で無効化可能
func (p *Provider) IsFunctionCallingEnabled() bool {
	return os.Getenv("DEEPSEEK_FUNCTION_CALLING") != "0"
}

// ChatWithTools は Provider interface の実装（context対応）
func (p *Provider) ChatWithTools(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	messages := openaicompat.BuildChatMessages(systemPrompt, history)

	// デバッグ: メッセージ構造をダンプ
	if os.Getenv("XELYON_DEBUG_DEEPSEEK") == "1" {
		errOut := api.ErrorWriterFromContext(ctx)
		fmt.Fprintf(errOut, "[DEBUG DeepSeek] === Messages (%d) ===\n", len(messages))
		for i, m := range messages {
			if i == 0 {
				fmt.Fprintf(errOut, "[DEBUG DeepSeek] [%d] role=%s (system, len=%d)\n", i, m.Role, len(m.Content))
				continue
			}
			tcIDs := make([]string, len(m.ToolCalls))
			for j, tc := range m.ToolCalls {
				tcIDs[j] = tc.ID
			}
			if len(tcIDs) > 0 {
				fmt.Fprintf(errOut, "[DEBUG DeepSeek] [%d] role=%s tool_calls=%v content_len=%d\n", i, m.Role, tcIDs, len(m.Content))
			} else if m.ToolCallID != "" {
				fmt.Fprintf(errOut, "[DEBUG DeepSeek] [%d] role=%s tool_call_id=%s\n", i, m.Role, m.ToolCallID)
			} else {
				fmt.Fprintf(errOut, "[DEBUG DeepSeek] [%d] role=%s content_len=%d\n", i, m.Role, len(m.Content))
			}
		}
	}

	// モデル名を設定（config優先、フォールバックは DeepSeek V4 Flash）
	requestedModel := api.GetDefaultModelWithContext(ctx, model, "deepseek", defaultDeepSeekModel)
	modelSelection := resolveDeepSeekModelSelection(ctx, requestedModel)
	extraFields, reasoningEffort, spinnerSuffix := deepSeekThinkingConfig(ctx, modelSelection)

	options := openaicompat.ChatCompletionsRequestOptions{
		Model:             modelSelection.actualModel,
		Messages:          messages,
		MaxTokens:         api.GetMaxOutputTokens(ctx, "deepseek", requestedModel),
		Stream:            true,
		IncludeUsage:      true,
		ReasoningEffort:   reasoningEffort,
		InitialToolChoice: "", // テスト期待値との整合性のため空文字列で初期化
		ExtraFields:       extraFields,
	}

	// Function Calling: ツール定義を追加（環境変数で無効化可能）
	if os.Getenv("DEEPSEEK_FUNCTION_CALLING") != "0" {
		options.FunctionCalling = &openaicompat.FunctionCallingOptions{
			Tools:    openai.GetCombinedOpenAIToolsWithContext(ctx, p.mcpTools),
			ToolName: p.toolChoice,
		}
	}

	reqBody := openaicompat.BuildChatCompletionsRequest(options)
	req, err := p.CreateAPIRequest(ctx, reqBody)
	if err != nil {
		return "", err
	}
	p.SetBearerAuth(req)

	return openaicompat.RunChatCompletions(ctx, p, req, openaicompat.ChatCompletionsRunOptions{
		SpinnerSuffix:      spinnerSuffix,
		ForceStreaming:     true,
		RequestErrorPrefix: "DeepSeek API request failed",
		StreamHandler:      p.handleStreamingResponse,
	})
}

// handleStreamingResponse はストリーミングレスポンスを処理（tool_calls対応）
func (p *Provider) handleStreamingResponse(ctx context.Context, resp *http.Response, spinner *ui.Spinner) (string, error) {
	out := api.OutputWriterFromContext(ctx)
	errOut := api.ErrorWriterFromContext(ctx)
	dim := color.New(color.Faint)
	reasoningActive := false

	// lastReasoningContent をリセット
	p.lastReasoningContent = ""

	streamResult, err := openaicompatstream.ParseSSEStream(ctx, resp, spinner, openaicompatstream.ParseSSEOptions{
		ValidateData: func(data string) error {
			if err := api.ValidateStreamResponse([]byte(data)); err != nil {
				return fmt.Errorf("invalid response structure: %w", err)
			}
			return nil
		},
		UsageDecoder: func(raw json.RawMessage) (*api.Usage, error) {
			if !openaicompatstream.HasUsagePayload(raw) {
				return nil, nil
			}
			var usagePayload struct {
				PromptTokens          int `json:"prompt_tokens"`
				CompletionTokens      int `json:"completion_tokens"`
				PromptCacheHitTokens  int `json:"prompt_cache_hit_tokens,omitempty"`
				PromptCacheMissTokens int `json:"prompt_cache_miss_tokens,omitempty"`
			}
			if err := json.Unmarshal(raw, &usagePayload); err != nil {
				return nil, err
			}

			usage := &api.Usage{
				InputTokens:       usagePayload.PromptTokens,
				OutputTokens:      usagePayload.CompletionTokens,
				CachedInputTokens: usagePayload.PromptCacheHitTokens,
			}
			if os.Getenv("XELYON_DEBUG_DEEPSEEK") == "1" {
				fmt.Fprintf(errOut, "[DEBUG DeepSeek] usage received: input=%d, output=%d, cached=%d\n",
					usagePayload.PromptTokens, usagePayload.CompletionTokens, usagePayload.PromptCacheHitTokens)
			}
			return usage, nil
		},
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
			// reasoning/content 表示後に tool_call へ切り替わる場合は spinner を再表示する。
			if !spinner.IsActive() {
				spinner.Start(ui.SpinnerMessageForTool(toolName))
			}
		},
		StopOnToolCallsFinish: true,
	})
	if err != nil {
		return "", err
	}

	// reasoning_content を保存（次のリクエストに含めるため）
	p.lastReasoningContent = streamResult.ReasoningContent

	// usage コールバックを呼び出し
	if streamResult.Usage != nil && p.usageCallback != nil {
		p.usageCallback(*streamResult.Usage)
	}

	toolCallsOutput := openaicompatstream.BuildToolCallJSON(
		streamResult.ToolCalls,
		openai.ConvertToolCallToToolJSON,
	)

	// tool_calls がある場合はそれを返す
	if toolCallsOutput != "" {
		if streamResult.Content != "" {
			return streamResult.Content + toolCallsOutput, nil
		}
		return toolCallsOutput, nil
	}
	return streamResult.Content, nil
}

// ChatWithImage は画像付きメッセージで会話を行う（非対応：テキストのみ送信）
func (p *Provider) ChatWithImage(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
	// DeepSeekは画像非対応なので警告を出してテキストのみ送信
	if image != nil && image.Base64 != "" {
		yellow.Fprintln(api.OutputWriterFromContext(ctx), "Warning: DeepSeek does not support image input. The image will be ignored.")
	}
	history = append(history, api.Message{Role: "user", Content: userMessage})
	return p.ChatWithTools(ctx, systemPrompt, history, model)
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

// LastReasoningContent は最後の API 呼び出しで返された reasoning_content を返す
// ReasoningContentProvider インターフェースの実装
func (p *Provider) LastReasoningContent() string {
	return p.lastReasoningContent
}
