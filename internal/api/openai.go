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

const defaultOpenAIURL = "https://api.openai.com/v1/chat/completions"
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

// OpenAIProvider はOpenAI APIのプロバイダー実装
type OpenAIProvider struct {
	apiKey     string
	apiURL     string
	httpClient *http.Client
}

// NewOpenAIProvider は新しいOpenAIProviderを作成
func NewOpenAIProvider(apiKey string) *OpenAIProvider {
	// 環境変数からURLをオーバーライド可能
	apiURL := os.Getenv("OPENAI_API_URL")
	if apiURL == "" {
		apiURL = defaultOpenAIURL
	}

	return &OpenAIProvider{
		apiKey: apiKey,
		apiURL: apiURL,
		httpClient: &http.Client{
			Timeout: config.DefaultHTTPTimeout,
		},
	}
}

// Name はプロバイダー名を返す
func (p *OpenAIProvider) Name() string {
	return "OpenAI"
}

// SupportsImages は画像入力対応を返す
func (p *OpenAIProvider) SupportsImages() bool {
	return true
}

// OpenAIContentPart はマルチモーダルコンテンツのパート
type OpenAIContentPart struct {
	Type     string          `json:"type"`                // "text" or "image_url"
	Text     string          `json:"text,omitempty"`      // type="text"の場合
	ImageURL *OpenAIImageURL `json:"image_url,omitempty"` // type="image_url"の場合
}

// OpenAIImageURL は画像URL
type OpenAIImageURL struct {
	URL string `json:"url"` // "data:image/png;base64,..." 形式
}

// OpenAIMultimodalMessage はマルチモーダルメッセージ
type OpenAIMultimodalMessage struct {
	Role    string              `json:"role"`
	Content []OpenAIContentPart `json:"content"`
}

// OpenAIMultimodalRequest はマルチモーダルAPIリクエスト
type OpenAIMultimodalRequest struct {
	Model           string        `json:"model"`
	Messages        []interface{} `json:"messages"` // Message or OpenAIMultimodalMessage
	Stream          bool          `json:"stream"`
	ReasoningEffort string        `json:"reasoning_effort,omitempty"` // low/medium/high
}

// levelToReasoningEffort は Thinking Level を OpenAI reasoning_effort に変換
func levelToReasoningEffort(level string) string {
	switch level {
	case "low":
		return "low"
	case "medium":
		return "medium"
	case "high", "xhigh":
		return "high"
	default:
		return "medium"
	}
}

// ChatWithTools は Provider interface の実装（context対応）
func (p *OpenAIProvider) ChatWithTools(ctx context.Context, systemPrompt string, history []Message, model string) (string, error) {
	if model == "" {
		model = "gpt-4o"
	}

	// モデルに応じて API を自動選択
	cfg := config.GetGlobalConfig()
	if cfg.IsResponsesAPIModel(model) {
		return p.chatWithResponses(ctx, systemPrompt, history, model)
	}
	return p.chatWithCompletions(ctx, systemPrompt, history, model)
}

// chatWithCompletions は Chat Completions API でチャット（既存実装）
func (p *OpenAIProvider) chatWithCompletions(ctx context.Context, systemPrompt string, history []Message, model string) (string, error) {
	cfg := config.GetGlobalConfig()

	// メッセージ構築
	messages := []Message{
		{Role: "system", Content: systemPrompt},
	}
	messages = append(messages, history...)

	reqBody := ChatRequest{
		Model:    model,
		Messages: messages,
		Stream:   true,
	}

	// Extended Thinking 適用
	if cfg.Thinking.Enabled {
		reqBody.ReasoningEffort = levelToReasoningEffort(cfg.Thinking.Level)
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
	spinner := ui.NewSpinner()
	spinnerMsg := "Thinking"
	if cfg.Thinking.Enabled {
		spinnerMsg = "Deep thinking"
	}
	spinner.Start(spinnerMsg)

	// 再利用可能なHTTPクライアントを使用
	resp, err := p.httpClient.Do(req)
	if err != nil {
		spinner.Stop()
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", HandleHTTPError(resp, spinner, p.Name())
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
func (p *OpenAIProvider) handleStreamingResponse(ctx context.Context, resp *http.Response, spinner *ui.Spinner) (string, error) {
	// OpenAI固有のパース処理
	parser := func(line string) (string, bool, error) {
		if !strings.HasPrefix(line, "data: ") {
			return "", false, nil
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			return "", true, nil
		}

		var streamResp StreamResponse
		if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
			return "", false, err
		}

		if len(streamResp.Choices) > 0 {
			return streamResp.Choices[0].Delta.Content, false, nil
		}

		return "", false, nil
	}

	return ParseStreamingResponse(ctx, resp, spinner, parser)
}

// handleNonStreamingResponse は非ストリーミングレスポンスを処理（フォールバック）
func (p *OpenAIProvider) handleNonStreamingResponse(resp *http.Response, spinner *ui.Spinner) (string, error) {
	return HandleNonStreamingResponse(resp, spinner)
}

// ChatWithImage は画像付きメッセージで会話を行う
func (p *OpenAIProvider) ChatWithImage(ctx context.Context, systemPrompt string, history []Message, userMessage string, image *ImageData, model string) (string, error) {
	// 画像がない場合は通常のChatWithToolsを使用
	if image == nil || image.Base64 == "" {
		history = append(history, Message{Role: "user", Content: userMessage})
		return p.ChatWithTools(ctx, systemPrompt, history, model)
	}

	// モデル名の設定（GPT-4V以降が必要）
	if model == "" {
		model = "gpt-4o"
	}

	// Responses API モデルの場合は専用の画像処理
	cfg := config.GetGlobalConfig()
	if cfg.IsResponsesAPIModel(model) {
		return p.chatWithImageResponses(ctx, systemPrompt, history, userMessage, image, model)
	}

	// システムプロンプトを最初のメッセージとして追加
	var messages []interface{}
	messages = append(messages, Message{Role: "system", Content: systemPrompt})

	// 履歴をメッセージ配列に追加（テキストのみ）
	for _, msg := range history {
		messages = append(messages, msg)
	}

	// Data URL形式で画像を埋め込む
	dataURL := fmt.Sprintf("data:%s;base64,%s", image.MediaType, image.Base64)

	// 画像付きユーザーメッセージを追加
	multimodalMessage := OpenAIMultimodalMessage{
		Role: "user",
		Content: []OpenAIContentPart{
			{
				Type: "image_url",
				ImageURL: &OpenAIImageURL{
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

	reqBody := OpenAIMultimodalRequest{
		Model:    model,
		Messages: messages,
		Stream:   true,
	}

	// Extended Thinking 適用
	if cfg.Thinking.Enabled {
		reqBody.ReasoningEffort = levelToReasoningEffort(cfg.Thinking.Level)
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
	spinner := ui.NewSpinner()
	spinnerMsg := "Analyzing image"
	if cfg.Thinking.Enabled {
		spinnerMsg = "Deep thinking (image)"
	}
	spinner.Start(spinnerMsg)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		spinner.Stop()
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", HandleHTTPError(resp, spinner, p.Name())
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
	spinner := ui.NewSpinner()
	spinnerMsg := "Thinking"
	if cfg.Thinking.Enabled {
		spinnerMsg = "Deep thinking"
	}
	spinner.Start(spinnerMsg)

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

// ResponsesResult は Responses API のレスポンス結果（ID付き）
type ResponsesResult struct {
	Content    string // テキストコンテンツ
	ResponseID string // レスポンスID（response.created から取得）
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
	spinner := ui.NewSpinner()
	spinnerMsg := "Analyzing image"
	if cfg.Thinking.Enabled {
		spinnerMsg = "Deep thinking (image)"
	}
	spinner.Start(spinnerMsg)

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
