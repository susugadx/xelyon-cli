package deepseek

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
	// メッセージ構築
	messages := []api.Message{
		{Role: "system", Content: systemPrompt},
	}
	messages = append(messages, history...)

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

	// モデル名を設定（config優先、フォールバックはdeepseek-chat）
	model = api.GetDefaultModelWithContext(ctx, model, "deepseek", "deepseek-chat")

	// Extended Thinking の ON/OFF でモデルを切り替え
	// DeepSeek は reasoner モデル自体が思考モードなので、モデル名で制御する
	if api.IsThinkingEnabled(ctx) {
		model = "deepseek-reasoner"
	} else if model == "deepseek-reasoner" {
		// /think off 時は deepseek-chat にフォールバック
		model = "deepseek-chat"
	}

	// モデル名マッピング
	actualModel := getActualModel(model)

	reqBody := api.ChatRequest{
		Model:         actualModel,
		Messages:      messages,
		MaxTokens:     api.GetMaxOutputTokens(ctx, "deepseek", model),
		Stream:        true,
		StreamOptions: &api.StreamOptions{IncludeUsage: true},
		ToolChoice:    "", // テスト期待値との整合性のため空文字列で初期化
	}

	// Function Calling: ツール定義を追加（環境変数で無効化可能）
	if os.Getenv("DEEPSEEK_FUNCTION_CALLING") != "0" {
		reqBody.Tools = openai.GetCombinedOpenAIToolsWithContext(ctx, p.mcpTools)
		reqBody.ToolChoice = "auto"

		// tool_choice 強制設定がある場合
		if p.toolChoice != nil {
			reqBody.ToolChoice = map[string]interface{}{
				"type": "function",
				"function": map[string]string{
					"name": *p.toolChoice,
				},
			}
		}
	}

	req, err := p.CreateAPIRequest(ctx, reqBody)
	if err != nil {
		return "", err
	}
	p.SetBearerAuth(req)

	// スピナー開始
	spinnerSuffix := ""
	if api.IsThinkingEnabled(ctx) {
		spinnerSuffix = "Reasoner"
	}
	spinner := api.StartThinkingSpinner(ctx, false, spinnerSuffix)

	// 再利用可能なHTTPクライアントを使用
	resp, err := p.ExecuteRequest(req)
	if err != nil {
		spinner.Stop()
		return "", fmt.Errorf("DeepSeek API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", api.HandleHTTPError(resp, spinner, p.Name())
	}

	// ストリーミング処理（tool_calls対応）
	return p.handleStreamingResponse(ctx, resp, spinner)
}

// handleStreamingResponse はストリーミングレスポンスを処理（tool_calls対応）
func (p *Provider) handleStreamingResponse(ctx context.Context, resp *http.Response, spinner *ui.Spinner) (string, error) {
	out := api.OutputWriterFromContext(ctx)
	errOut := api.ErrorWriterFromContext(ctx)
	toolCalls := openaicompatstream.NewToolCallCollector()

	// reasoning_content を累積
	var reasoningContent strings.Builder
	reasoningStarted := false

	// usage 情報を追跡
	var lastUsage *api.Usage

	dim := color.New(color.Faint)

	// DeepSeek固有のパース処理（OpenAI互換形式）
	parser := func(line string) (string, bool, error) {
		data, done, handled := openaicompatstream.ParseSSEDataLine(line)
		if !handled {
			return "", false, nil
		}
		if done {
			return "", true, nil
		}

		// レスポンス構造を検証
		if err := api.ValidateStreamResponse([]byte(data)); err != nil {
			return "", false, fmt.Errorf("invalid response structure: %w", err)
		}

		chunk, err := openaicompatstream.DecodeChunk(data)
		if err != nil {
			return "", false, err
		}

		if openaicompatstream.HasUsagePayload(chunk.Usage) {
			var usagePayload struct {
				PromptTokens          int `json:"prompt_tokens"`
				CompletionTokens      int `json:"completion_tokens"`
				PromptCacheHitTokens  int `json:"prompt_cache_hit_tokens,omitempty"`
				PromptCacheMissTokens int `json:"prompt_cache_miss_tokens,omitempty"`
			}
			if err := json.Unmarshal(chunk.Usage, &usagePayload); err != nil {
				return "", false, err
			}

			lastUsage = &api.Usage{
				InputTokens:       usagePayload.PromptTokens,
				OutputTokens:      usagePayload.CompletionTokens,
				CachedInputTokens: usagePayload.PromptCacheHitTokens,
			}
			if os.Getenv("XELYON_DEBUG_DEEPSEEK") == "1" {
				fmt.Fprintf(errOut, "[DEBUG DeepSeek] usage received: input=%d, output=%d, cached=%d\n",
					usagePayload.PromptTokens, usagePayload.CompletionTokens, usagePayload.PromptCacheHitTokens)
			}
		}

		if len(chunk.Choices) == 0 {
			return "", false, nil
		}

		choice := chunk.Choices[0]

		// reasoning_content の累積・表示
		if choice.Delta.ReasoningContent != "" {
			if !reasoningStarted {
				reasoningStarted = true
				// スピナーを停止して思考表示開始
				spinner.Stop()
				dim.Fprint(out, "💭 ")
			}
			reasoningContent.WriteString(choice.Delta.ReasoningContent)
			dim.Fprint(out, choice.Delta.ReasoningContent)
		}

		// reasoning_content から content に切り替わった時に改行
		if choice.Delta.Content != "" && reasoningStarted && reasoningContent.Len() > 0 {
			_, _ = fmt.Fprintln(out) // 思考内容の後に改行
			_, _ = fmt.Fprintln(out) // 空行で区切り
			reasoningStarted = false
		}

		toolCalls.Append(choice.Delta.ToolCalls, func(toolName string) {
			// reasoning/content 表示後に tool_call へ切り替わる場合は spinner を再表示する。
			if !spinner.IsActive() {
				spinner.Start(ui.SpinnerMessageForTool(toolName))
			}
		})

		// finish_reason == "tool_calls" で完了
		if choice.FinishReason == "tool_calls" {
			// 思考のみで終了した場合の改行
			if reasoningStarted && reasoningContent.Len() > 0 {
				_, _ = fmt.Fprintln(out)
				_, _ = fmt.Fprintln(out)
			}
			return "", true, nil
		}

		// テキストコンテンツ
		return choice.Delta.Content, false, nil
	}

	// lastReasoningContent をリセット
	p.lastReasoningContent = ""

	content, err := api.ParseStreamingResponse(ctx, resp, spinner, parser)
	if err != nil {
		return "", err
	}

	// reasoning_content を保存（次のリクエストに含めるため）
	if reasoningContent.Len() > 0 {
		p.lastReasoningContent = reasoningContent.String()
	}

	// usage コールバックを呼び出し
	if lastUsage != nil && p.usageCallback != nil {
		p.usageCallback(*lastUsage)
	}

	toolCallsOutput := openaicompatstream.BuildToolCallJSON(
		toolCalls.ToOpenAIToolCalls(),
		openai.ConvertToolCallToToolJSON,
	)

	// tool_calls がある場合はそれを返す
	if toolCallsOutput != "" {
		if content != "" {
			return content + toolCallsOutput, nil
		}
		return toolCallsOutput, nil
	}
	return content, nil
}

// getActualModel はモデル名を実際のAPI用に変換
func getActualModel(model string) string {
	switch model {
	case "deepseek-chat", "":
		return "deepseek-chat"
	case "deepseek-coder":
		return "deepseek-coder"
	case "deepseek-reasoner":
		return "deepseek-reasoner"
	default:
		return "deepseek-chat"
	}
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
