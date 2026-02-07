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
	"github.com/susugadx/xelyon-cli/internal/config"
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
	cfg := config.GetGlobalConfig()

	// メッセージ構築
	messages := []api.Message{
		{Role: "system", Content: systemPrompt},
	}
	messages = append(messages, history...)

	reqBody := api.ChatRequest{
		Model:                model,
		Messages:             messages,
		MaxTokens:            api.GetMaxOutputTokens("openai", model),
		Stream:               true,
		StreamOptions:        &api.StreamOptions{IncludeUsage: true},
		PromptCacheKey:       "xelyon",
		PromptCacheRetention: "24h",
	}

	// Function Calling: ツール定義を追加（環境変数で無効化可能）
	if os.Getenv("OPENAI_FUNCTION_CALLING") != "0" {
		reqBody.Tools = GetCombinedOpenAITools(p.mcpTools)
		reqBody.ToolChoice = "auto"
	}

	// Extended Thinking 適用
	if cfg.Thinking.Enabled {
		reqBody.ReasoningEffort = LevelToReasoningEffort(cfg.Thinking.Level)
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	// スピナー開始
	spinner := api.StartThinkingSpinner(false, "")

	// 再利用可能なHTTPクライアントを使用
	resp, err := p.httpClient.Do(req)
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
		return p.handleNonStreamingResponse(resp, spinner)
	}
}

// toolCallAccumulator は分割された tool_calls を累積するための構造体
type toolCallAccumulator struct {
	ID        string
	Name      string
	Arguments strings.Builder
}

// handleStreamingResponse はストリーミングレスポンスを処理（tool_calls対応）
func (p *Provider) handleStreamingResponse(ctx context.Context, resp *http.Response, spinner *ui.Spinner) (string, error) {
	// ストリーミングで分割されて送られてくる tool_calls を累積
	toolCalls := make(map[int]*toolCallAccumulator)
	var toolCallsOutput strings.Builder

	// usage 情報を追跡
	var lastUsage *api.Usage

	// OpenAI固有のパース処理
	parser := func(line string) (string, bool, error) {
		if !strings.HasPrefix(line, "data: ") {
			return "", false, nil
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			return "", true, nil
		}

		// 拡張した StreamResponse（tool_calls, finish_reason, usage を含む）
		var streamResp struct {
			Choices []struct {
				Delta struct {
					Content   string               `json:"content,omitempty"`
					ToolCalls []api.OpenAIToolCall `json:"tool_calls,omitempty"`
				} `json:"delta"`
				FinishReason string `json:"finish_reason,omitempty"`
			} `json:"choices"`
			Usage *struct {
				PromptTokens        int `json:"prompt_tokens"`
				CompletionTokens    int `json:"completion_tokens"`
				PromptTokensDetails *struct {
					CachedTokens int `json:"cached_tokens,omitempty"`
				} `json:"prompt_tokens_details,omitempty"`
			} `json:"usage,omitempty"`
		}
		if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
			return "", false, err
		}

		// usage 情報を記録
		if streamResp.Usage != nil {
			cachedTokens := 0
			if streamResp.Usage.PromptTokensDetails != nil {
				cachedTokens = streamResp.Usage.PromptTokensDetails.CachedTokens
			}
			lastUsage = &api.Usage{
				InputTokens:       streamResp.Usage.PromptTokens,
				OutputTokens:      streamResp.Usage.CompletionTokens,
				CachedInputTokens: cachedTokens,
			}
			if os.Getenv("XELYON_DEBUG_OPENAI") == "1" {
				fmt.Fprintf(os.Stderr, "[DEBUG OpenAI] usage received: input=%d, output=%d, cached=%d\n",
					streamResp.Usage.PromptTokens, streamResp.Usage.CompletionTokens, cachedTokens)
			}
		}

		if len(streamResp.Choices) == 0 {
			return "", false, nil
		}

		choice := streamResp.Choices[0]

		// tool_calls の累積処理
		for _, tc := range choice.Delta.ToolCalls {
			acc, exists := toolCalls[tc.Index]
			if !exists {
				acc = &toolCallAccumulator{}
				toolCalls[tc.Index] = acc
			}
			if tc.ID != "" {
				acc.ID = tc.ID
			}
			if tc.Function.Name != "" {
				acc.Name = tc.Function.Name
			}
			if tc.Function.Arguments != "" {
				acc.Arguments.WriteString(tc.Function.Arguments)
			}
		}

		// finish_reason == "tool_calls" で完了
		if choice.FinishReason == "tool_calls" {
			// 累積した tool_calls を内部JSON形式に変換
			for i := 0; i < len(toolCalls); i++ {
				acc := toolCalls[i]
				if acc == nil {
					continue
				}
				tc := &api.OpenAIToolCall{
					ID:   acc.ID,
					Type: "function",
					Function: api.OpenAIToolCallFunction{
						Name:      acc.Name,
						Arguments: acc.Arguments.String(),
					},
				}
				if toolJSON, err := ConvertToolCallToToolJSON(tc); err == nil {
					toolCallsOutput.WriteString(toolJSON)
				}
			}
			return "", true, nil
		}

		// テキストコンテンツ
		return choice.Delta.Content, false, nil
	}

	content, err := api.ParseStreamingResponse(ctx, resp, spinner, parser)
	if err != nil {
		return "", err
	}

	// usage コールバックを呼び出し
	if lastUsage != nil && p.usageCallback != nil {
		p.usageCallback(*lastUsage)
	}

	// tool_calls がある場合はそれを返す
	if toolCallsOutput.Len() > 0 {
		if content != "" {
			return content + toolCallsOutput.String(), nil
		}
		return toolCallsOutput.String(), nil
	}
	return content, nil
}

// handleNonStreamingResponse は非ストリーミングレスポンスを処理（フォールバック）
func (p *Provider) handleNonStreamingResponse(resp *http.Response, spinner *ui.Spinner) (string, error) {
	return api.HandleNonStreamingResponse(resp, spinner)
}

// chatWithImageCompletions は Completions API で画像付きメッセージを処理
func (p *Provider) chatWithImageCompletions(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
	cfg := config.GetGlobalConfig()

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
		MaxTokens:            api.GetMaxOutputTokens("openai", model),
		Stream:               true,
		PromptCacheKey:       "xelyon",
		PromptCacheRetention: "24h",
	}

	// Extended Thinking 適用
	if cfg.Thinking.Enabled {
		reqBody.ReasoningEffort = LevelToReasoningEffort(cfg.Thinking.Level)
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	// スピナー開始
	spinner := api.StartThinkingSpinner(true, "")

	resp, err := p.httpClient.Do(req)
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
		return p.handleNonStreamingResponse(resp, spinner)
	}
}
