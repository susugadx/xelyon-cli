package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
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
	Model           string        `json:"model"`
	Messages        []interface{} `json:"messages"` // Message or MultimodalMessage
	Stream          bool          `json:"stream"`
	ReasoningEffort string        `json:"reasoning_effort,omitempty"` // low/medium/high
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
		Model:    model,
		Messages: messages,
		Stream:   true,
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

// handleStreamingResponse はストリーミングレスポンスを処理
func (p *Provider) handleStreamingResponse(ctx context.Context, resp *http.Response, spinner *ui.Spinner) (string, error) {
	// OpenAI固有のパース処理
	parser := func(line string) (string, bool, error) {
		if !strings.HasPrefix(line, "data: ") {
			return "", false, nil
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			return "", true, nil
		}

		var streamResp api.StreamResponse
		if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
			return "", false, err
		}

		if len(streamResp.Choices) > 0 {
			return streamResp.Choices[0].Delta.Content, false, nil
		}

		return "", false, nil
	}

	return api.ParseStreamingResponse(ctx, resp, spinner, parser)
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
		Model:    model,
		Messages: messages,
		Stream:   true,
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
