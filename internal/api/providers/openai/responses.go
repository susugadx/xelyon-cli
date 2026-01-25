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

const defaultOpenAIResponsesURL = "https://api.openai.com/v1/responses"

// ReasoningConfig は OpenAI Extended Thinking の設定
type ReasoningConfig struct {
	Effort string `json:"effort"` // none, low, medium, high（omitempty削除で明示的に送信）
}

// ResponsesTool は Responses API 用のツール定義
// Chat Completions API と異なり、"function" キーのネストが不要
type ResponsesTool struct {
	Type        string                 `json:"type"`                  // "function"
	Name        string                 `json:"name"`                  // ツール名
	Description string                 `json:"description,omitempty"` // ツールの説明
	Parameters  map[string]interface{} `json:"parameters,omitempty"`  // JSON Schema
	Strict      bool                   `json:"strict,omitempty"`      // Structured Output
}

// ResponsesRequest は Responses API リクエスト
type ResponsesRequest struct {
	Model              string           `json:"model"`
	Input              interface{}      `json:"input,omitempty"`                // string or []InputItem（previous_response_id使用時は省略可）
	PreviousResponseID string           `json:"previous_response_id,omitempty"` // 前回のレスポンスID（キャッシュ用）
	Instructions       string           `json:"instructions,omitempty"`         // システムプロンプト
	Stream             bool             `json:"stream,omitempty"`
	Reasoning          *ReasoningConfig `json:"reasoning,omitempty"` // Extended Thinking
	Tools              []ResponsesTool  `json:"tools,omitempty"`     // ツール定義
}

// InputItem は api.InputItem のエイリアス（api packageで定義）
type InputItem = api.InputItem

// InputContentPart は api.InputContentPart のエイリアス（api packageで定義）
type InputContentPart = api.InputContentPart

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
// previous_response_id を使用してキャッシュを活用
func (p *Provider) chatWithResponses(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	cfg := config.GetGlobalConfig()

	// Responses API URL
	apiURL := os.Getenv("OPENAI_RESPONSES_URL")
	if apiURL == "" {
		apiURL = defaultOpenAIResponsesURL
	}

	reqBody := ResponsesRequest{
		Model:        model,
		Instructions: systemPrompt,
		Stream:       true,
	}

	// previous_response_id がある場合はキャッシュを活用
	if p.lastResponseID != "" && len(history) > 0 {
		// 最新のユーザーメッセージのみ送信
		lastMsg := history[len(history)-1]
		reqBody.PreviousResponseID = p.lastResponseID
		reqBody.Input = []InputItem{{
			Type:    "message",
			Role:    lastMsg.Role,
			Content: lastMsg.Content,
		}}
	} else {
		// 初回または responseID がない場合は履歴全体を送信
		var input []InputItem
		for _, msg := range history {
			input = append(input, InputItem{
				Type:    "message",
				Role:    msg.Role,
				Content: msg.Content,
			})
		}
		reqBody.Input = input
	}

	// Extended Thinking 適用（明示的に設定）
	if cfg.Thinking.Enabled {
		reqBody.Reasoning = &ReasoningConfig{
			Effort: LevelToReasoningEffort(cfg.Thinking.Level),
		}
	} else {
		// 明示的に無効化（GPT-5.2でもreasoningが勝手に有効になる問題を回避）
		reqBody.Reasoning = &ReasoningConfig{
			Effort: "none",
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
	spinner := api.StartThinkingSpinner(false, "")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		spinner.Stop()
		return "", fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	// previous_response_id が無効な場合のフォールバック
	if resp.StatusCode == http.StatusBadRequest && p.lastResponseID != "" {
		// responseID をクリアしてリトライ
		p.lastResponseID = ""
		resp.Body.Close()
		return p.chatWithResponses(ctx, systemPrompt, history, model)
	}

	if resp.StatusCode != http.StatusOK {
		return "", api.HandleHTTPError(resp, spinner, p.Name())
	}

	content, responseID, err := p.handleResponsesStreaming(ctx, resp, spinner)
	if err == nil && responseID != "" {
		p.lastResponseID = responseID
	}
	return content, err
}

// handleResponsesStreaming は Responses API のストリーミングを処理
// Response ID も抽出して返却（content, responseID, error）
func (p *Provider) handleResponsesStreaming(ctx context.Context, resp *http.Response, spinner *ui.Spinner) (string, string, error) {
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

	content, err := api.ParseStreamingResponse(ctx, resp, spinner, parser)
	return content, responseID, err
}

// chatWithImageResponses は Responses API で画像付きメッセージを処理
// NOTE: 画像付きの場合は previous_response_id を使用しない（キャッシュ動作が不明瞭なため）
func (p *Provider) chatWithImageResponses(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
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

	// Extended Thinking 適用（明示的に設定）
	if cfg.Thinking.Enabled {
		reqBody.Reasoning = &ReasoningConfig{
			Effort: LevelToReasoningEffort(cfg.Thinking.Level),
		}
	} else {
		// 明示的に無効化
		reqBody.Reasoning = &ReasoningConfig{
			Effort: "none",
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
	spinner := api.StartThinkingSpinner(true, "")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		spinner.Stop()
		return "", fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", api.HandleHTTPError(resp, spinner, p.Name())
	}

	content, responseID, err := p.handleResponsesStreaming(ctx, resp, spinner)
	if err == nil && responseID != "" {
		// 画像メッセージ後も responseID を保存（次回テキストのみの場合に使用可能）
		p.lastResponseID = responseID
	}
	return content, err
}
