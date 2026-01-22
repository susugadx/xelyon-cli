package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

const defaultOpenAIResponsesURL = "https://api.openai.com/v1/responses"

// ReasoningConfig は OpenAI Extended Thinking の設定
type ReasoningConfig struct {
	Effort string `json:"effort,omitempty"` // low, medium, high
}

// ResponsesRequest は Responses API リクエスト
type ResponsesRequest struct {
	Model        string           `json:"model"`
	Input        interface{}      `json:"input"`                  // string or []InputItem
	Instructions string           `json:"instructions,omitempty"` // システムプロンプト
	Stream       bool             `json:"stream,omitempty"`
	Reasoning    *ReasoningConfig `json:"reasoning,omitempty"` // Extended Thinking
}

// InputItem は Responses API の入力アイテム（Compact API対応拡張版）
// ユーザーメッセージ、アシスタント応答、圧縮済みアイテムを表現
type InputItem struct {
	Type    string      `json:"type"`              // "message" or "compacted"
	Role    string      `json:"role,omitempty"`    // "user", "assistant"
	Content interface{} `json:"content,omitempty"` // string or []InputContentPart

	// アシスタント応答の完全情報（Compact API用）
	ID     string `json:"id,omitempty"`     // "msg_xxx" (アシスタント応答のID)
	Status string `json:"status,omitempty"` // "completed"

	// 圧縮済みアイテム用
	Data string `json:"data,omitempty"` // 暗号化データ（type="compacted"の場合）
}

// InputContentPart は Responses API のコンテンツパート（画像対応）
type InputContentPart struct {
	Type     string `json:"type"`                // "input_text" or "input_image"
	Text     string `json:"text,omitempty"`      // type="input_text"の場合
	ImageURL string `json:"image_url,omitempty"` // type="input_image"の場合（data:image/...形式）
}

// ResponseMetadata はレスポンスメタデータ（response.created イベント用）
type ResponseMetadata struct {
	ID     string `json:"id"`               // "resp_xxx..."
	Status string `json:"status,omitempty"` // "in_progress", "completed"
	Model  string `json:"model,omitempty"`
}

// ResponsesStreamChunk は Responses API ストリーミングチャンク
type ResponsesStreamChunk struct {
	Type     string            `json:"type"`               // "response.output_text.delta", "response.created", etc.
	Delta    string            `json:"delta,omitempty"`    // テキスト差分
	Response *ResponseMetadata `json:"response,omitempty"` // response.created で取得
}

// ResponsesResult は Responses API のレスポンス結果（ID付き）
type ResponsesResult struct {
	Content    string // テキストコンテンツ
	ResponseID string // レスポンスID（response.created から取得）
}

// chatWithResponses は Responses API でチャット
func (p *OpenAIProvider) chatWithResponses(ctx context.Context, systemPrompt string, history []Message, model string) (string, error) {
	cfg := config.GetGlobalConfig()

	// Responses API URL
	apiURL := os.Getenv("OPENAI_RESPONSES_URL")
	if apiURL == "" {
		apiURL = defaultOpenAIResponsesURL
	}

	// 入力を構築
	var input []InputItem
	for _, msg := range history {
		input = append(input, InputItem{
			Type:    "message",
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	reqBody := ResponsesRequest{
		Model:        model,
		Input:        input,
		Instructions: systemPrompt,
		Stream:       true,
	}

	// Extended Thinking 適用
	if cfg.Thinking.Enabled {
		reqBody.Reasoning = &ReasoningConfig{
			Effort: levelToReasoningEffort(cfg.Thinking.Level),
		}
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	// スピナー開始
	spinner := StartThinkingSpinner(false, "")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		spinner.Stop()
		return "", fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", HandleHTTPError(resp, spinner, p.Name())
	}

	return p.handleResponsesStreaming(ctx, resp, spinner)
}

// handleResponsesStreaming は Responses API のストリーミングを処理
// Response ID も抽出して返却
func (p *OpenAIProvider) handleResponsesStreaming(ctx context.Context, resp *http.Response, spinner *ui.Spinner) (string, error) {
	var responseID string // response.created イベントから抽出

	parser := func(line string) (string, bool, error) {
		// SSE形式: "event: xxx" と "data: {...}" の組み合わせ
		if !strings.HasPrefix(line, "data: ") {
			return "", false, nil
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			return "", true, nil
		}

		var chunk ResponsesStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return "", false, nil // パースエラーはスキップ
		}

		// response.created イベントから Response ID を抽出
		if chunk.Type == "response.created" && chunk.Response != nil {
			responseID = chunk.Response.ID
		}

		// response.output_text.delta でテキスト差分を取得
		if chunk.Type == "response.output_text.delta" {
			return chunk.Delta, false, nil
		}

		// response.completed または response.done で終了
		if chunk.Type == "response.completed" || chunk.Type == "response.done" {
			return "", true, nil
		}

		return "", false, nil
	}

	content, err := ParseStreamingResponse(ctx, resp, spinner, parser)
	// NOTE: responseID は現在使用されていないが、将来の Compact API 統合で使用予定
	_ = responseID
	return content, err
}

// chatWithImageResponses は Responses API で画像付きメッセージを処理
func (p *OpenAIProvider) chatWithImageResponses(ctx context.Context, systemPrompt string, history []Message, userMessage string, image *ImageData, model string) (string, error) {
	cfg := config.GetGlobalConfig()

	// Responses API URL
	apiURL := os.Getenv("OPENAI_RESPONSES_URL")
	if apiURL == "" {
		apiURL = defaultOpenAIResponsesURL
	}

	// 入力を構築
	var input []InputItem

	// 履歴を追加
	for _, msg := range history {
		input = append(input, InputItem{
			Type:    "message",
			Role:    msg.Role,
			Content: msg.Content,
		})
	}

	// 画像付きユーザーメッセージを追加
	dataURL := fmt.Sprintf("data:%s;base64,%s", image.MediaType, image.Base64)
	imageMessage := InputItem{
		Type: "message",
		Role: "user",
		Content: []InputContentPart{
			{
				Type:     "input_image",
				ImageURL: dataURL,
			},
			{
				Type: "input_text",
				Text: userMessage,
			},
		},
	}
	input = append(input, imageMessage)

	reqBody := ResponsesRequest{
		Model:        model,
		Input:        input,
		Instructions: systemPrompt,
		Stream:       true,
	}

	// Extended Thinking 適用
	if cfg.Thinking.Enabled {
		reqBody.Reasoning = &ReasoningConfig{
			Effort: levelToReasoningEffort(cfg.Thinking.Level),
		}
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	// スピナー開始
	spinner := StartThinkingSpinner(true, "")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		spinner.Stop()
		return "", fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", HandleHTTPError(resp, spinner, p.Name())
	}

	return p.handleResponsesStreaming(ctx, resp, spinner)
}
