package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	openaicompatstream "github.com/susugadx/xelyon-cli/internal/api/providers/openai_compat_stream"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// ContentPart はマルチモーダルコンテンツのパート
type ContentPart struct {
	Type     string    `json:"type"`                // "text" or "image_url"
	Text     string    `json:"text,omitempty"`      // type="text"の場合
	ImageURL *ImageURL `json:"image_url,omitempty"` // type="image_url"の場合
}

// ImageURL は画像URL
type ImageURL struct {
	URL string `json:"url"` // "data:image/png;base64,..." 形式
}

// MultimodalMessage はマルチモーダルメッセージ
type MultimodalMessage struct {
	Role    string        `json:"role"`
	Content []ContentPart `json:"content"`
}

// MultimodalRequest はマルチモーダルAPIリクエスト
type MultimodalRequest struct {
	Model                string        `json:"model"`
	Messages             []interface{} `json:"messages"`             // Message or MultimodalMessage
	MaxTokens            int           `json:"max_tokens,omitempty"` // 最大出力トークン数
	Stream               bool          `json:"stream"`
	ReasoningEffort      string        `json:"reasoning_effort,omitempty"`       // low/medium/high
	PromptCacheKey       string        `json:"prompt_cache_key,omitempty"`       // プロンプトキャッシュのルーティングキー
	PromptCacheRetention string        `json:"prompt_cache_retention,omitempty"` // キャッシュ保持期間（"24h"でextended cache）
}

// chatWithCompletions は Chat Completions API でチャット
func (p *Provider) chatWithCompletions(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	cfg := tools.ConfigFromContext(ctx)

	// メッセージ構築
	messages := []api.Message{
		{Role: "system", Content: systemPrompt},
	}
	messages = append(messages, history...)

	reqBody := api.ChatRequest{
		Model:                model,
		Messages:             messages,
		MaxTokens:            api.GetMaxOutputTokens(ctx, "openai", model),
		Stream:               true,
		StreamOptions:        &api.StreamOptions{IncludeUsage: true},
		PromptCacheKey:       BuildPromptCacheKey(model, systemPrompt),
		PromptCacheRetention: "24h",
	}

	// Function Calling: ツール定義を追加（環境変数で無効化可能）
	if os.Getenv("OPENAI_FUNCTION_CALLING") != "0" {
		reqBody.Tools = GetCombinedOpenAIToolsWithContext(ctx, p.mcpTools)
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

	// Extended Thinking 適用
	if api.IsThinkingEnabled(ctx) {
		reqBody.ReasoningEffort = LevelToReasoningEffort(cfg.Thinking.Level)
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.APIURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.APIKey)

	// スピナー開始
	spinner := api.StartThinkingSpinner(ctx, false, "")

	// 再利用可能なHTTPクライアントを使用
	resp, err := p.ExecuteRequest(req)
	if err != nil {
		spinner.Stop()
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", api.HandleHTTPError(resp, spinner, p.Name())
	}

	// Content-Typeでストリーミング対応を判定
	contentType := resp.Header.Get("Content-Type")
	isStreaming := strings.Contains(contentType, "text/event-stream")

	if isStreaming {
		return p.handleStreamingResponse(ctx, resp, spinner)
	} else {
		return p.handleNonStreamingResponse(ctx, resp, spinner)
	}
}

// handleStreamingResponse はストリーミングレスポンスを処理（tool_calls対応）
func (p *Provider) handleStreamingResponse(ctx context.Context, resp *http.Response, spinner *ui.Spinner) (string, error) {
	streamResult, err := openaicompatstream.ParseSSEStream(ctx, resp, spinner, openaicompatstream.ParseSSEOptions{
		UsageDecoder:          decodeOpenAICompatUsage,
		StopOnToolCallsFinish: true,
		OnToolCallArguments: func(toolName string) {
			// tool_call arguments が届いた時のみ spinner を tool 名で再表示する。
			if !spinner.IsActive() {
				spinner.Start(ui.SpinnerMessageForTool(toolName))
			}
		},
	})
	if err != nil {
		return "", err
	}

	// usage コールバックを呼び出し
	if streamResult.Usage != nil {
		if os.Getenv("XELYON_DEBUG_OPENAI") == "1" {
			fmt.Fprintf(api.ErrorWriterFromContext(ctx), "[DEBUG OpenAI] usage received: input=%d, output=%d, cached=%d\n",
				streamResult.Usage.InputTokens, streamResult.Usage.OutputTokens, streamResult.Usage.CachedInputTokens)
		}
		if p.usageCallback != nil {
			p.usageCallback(*streamResult.Usage)
		}
	}

	toolCallsOutput := openaicompatstream.BuildToolCallJSON(
		streamResult.ToolCalls,
		ConvertToolCallToToolJSON,
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

func decodeOpenAICompatUsage(raw json.RawMessage) (*api.Usage, error) {
	if !openaicompatstream.HasUsagePayload(raw) {
		return nil, nil
	}

	var usagePayload struct {
		PromptTokens        int `json:"prompt_tokens"`
		CompletionTokens    int `json:"completion_tokens"`
		PromptTokensDetails *struct {
			CachedTokens int `json:"cached_tokens,omitempty"`
		} `json:"prompt_tokens_details,omitempty"`
		CompletionTokensDetails *struct {
			ReasoningTokens int `json:"reasoning_tokens,omitempty"`
		} `json:"completion_tokens_details,omitempty"`
	}
	if err := json.Unmarshal(raw, &usagePayload); err != nil {
		return nil, err
	}

	cachedTokens := 0
	if usagePayload.PromptTokensDetails != nil {
		cachedTokens = usagePayload.PromptTokensDetails.CachedTokens
	}
	reasoningTokens := 0
	if usagePayload.CompletionTokensDetails != nil {
		reasoningTokens = usagePayload.CompletionTokensDetails.ReasoningTokens
	}

	return &api.Usage{
		InputTokens:       usagePayload.PromptTokens,
		OutputTokens:      usagePayload.CompletionTokens,
		ThinkingTokens:    reasoningTokens,
		CachedInputTokens: cachedTokens,
	}, nil
}

// handleNonStreamingResponse は非ストリーミングレスポンスを処理（フォールバック）
func (p *Provider) handleNonStreamingResponse(ctx context.Context, resp *http.Response, spinner *ui.Spinner) (string, error) {
	return api.HandleNonStreamingResponse(ctx, resp, spinner)
}

// chatWithImageCompletions は Completions API で画像付きメッセージを処理
func (p *Provider) chatWithImageCompletions(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
	cfg := tools.ConfigFromContext(ctx)

	// システムプロンプトを最初のメッセージとして追加
	var messages []interface{}
	messages = append(messages, api.Message{Role: "system", Content: systemPrompt})

	// 履歴をメッセージ配列に追加（テキストのみ）
	for _, msg := range history {
		messages = append(messages, msg)
	}

	// Data URL形式で画像を埋め込む
	dataURL := fmt.Sprintf("data:%s;base64,%s", image.MediaType, image.Base64)

	// 画像付きユーザーメッセージを追加
	multimodalMessage := MultimodalMessage{
		Role: "user",
		Content: []ContentPart{
			{
				Type: "image_url",
				ImageURL: &ImageURL{
					URL: dataURL,
				},
			},
			{
				Type: "text",
				Text: userMessage,
			},
		},
	}
	messages = append(messages, multimodalMessage)

	reqBody := MultimodalRequest{
		Model:                model,
		Messages:             messages,
		MaxTokens:            api.GetMaxOutputTokens(ctx, "openai", model),
		Stream:               true,
		PromptCacheKey:       BuildPromptCacheKey(model, systemPrompt),
		PromptCacheRetention: "24h",
	}

	// Extended Thinking 適用
	if api.IsThinkingEnabled(ctx) {
		reqBody.ReasoningEffort = LevelToReasoningEffort(cfg.Thinking.Level)
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.APIURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.APIKey)

	// スピナー開始
	spinner := api.StartThinkingSpinner(ctx, true, "")

	resp, err := p.ExecuteRequest(req)
	if err != nil {
		spinner.Stop()
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", api.HandleHTTPError(resp, spinner, p.Name())
	}

	// ストリーミング処理
	contentType := resp.Header.Get("Content-Type")
	isStreaming := strings.Contains(contentType, "text/event-stream")

	if isStreaming {
		return p.handleStreamingResponse(ctx, resp, spinner)
	} else {
		return p.handleNonStreamingResponse(ctx, resp, spinner)
	}
}
