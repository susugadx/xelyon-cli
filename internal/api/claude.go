package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

const defaultClaudeURL = "https://api.anthropic.com/v1/messages"

// ClaudeProvider はClaude (Anthropic) APIのプロバイダー実装
type ClaudeProvider struct {
	apiKey     string
	apiURL     string
	httpClient *http.Client
}

// NewClaudeProvider は新しいClaudeProviderを作成
func NewClaudeProvider(apiKey string) *ClaudeProvider {
	// 環境変数からURLをオーバーライド可能
	apiURL := os.Getenv("ANTHROPIC_API_URL")
	if apiURL == "" {
		apiURL = defaultClaudeURL
	}

	return &ClaudeProvider{
		apiKey: apiKey,
		apiURL: apiURL,
		httpClient: &http.Client{
			Timeout: config.DefaultHTTPTimeout,
		},
	}
}

// Name はプロバイダー名を返す
func (p *ClaudeProvider) Name() string {
	return "Claude"
}

// SupportsImages は画像入力対応を返す
func (p *ClaudeProvider) SupportsImages() bool {
	return true
}

// ClaudeMessage はClaudeのメッセージ構造
type ClaudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ClaudeRequest はClaude APIリクエスト
type ClaudeRequest struct {
	Model     string          `json:"model"`
	Messages  []ClaudeMessage `json:"messages"`
	System    string          `json:"system,omitempty"`
	MaxTokens int             `json:"max_tokens"`
	Stream    bool            `json:"stream"`
}

// ClaudeDelta はストリームの差分
type ClaudeDelta struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ClaudeStreamEvent はストリームイベント
type ClaudeStreamEvent struct {
	Type  string      `json:"type"`
	Delta ClaudeDelta `json:"delta"`
}

// ClaudeContent はレスポンスのコンテンツ
type ClaudeContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ClaudeContentPart はマルチモーダルコンテンツのパート
type ClaudeContentPart struct {
	Type   string             `json:"type"`             // "text" or "image"
	Text   string             `json:"text,omitempty"`   // type="text"の場合
	Source *ClaudeImageSource `json:"source,omitempty"` // type="image"の場合
}

// ClaudeImageSource は画像ソース
type ClaudeImageSource struct {
	Type      string `json:"type"`       // "base64"
	MediaType string `json:"media_type"` // "image/png", "image/jpeg" etc
	Data      string `json:"data"`       // Base64エンコードされたデータ
}

// ClaudeMultimodalMessage はマルチモーダルメッセージ（画像含む）
type ClaudeMultimodalMessage struct {
	Role    string              `json:"role"`
	Content []ClaudeContentPart `json:"content"`
}

// ClaudeMultimodalRequest はマルチモーダルAPIリクエスト
type ClaudeMultimodalRequest struct {
	Model     string        `json:"model"`
	Messages  []interface{} `json:"messages"` // ClaudeMessage or ClaudeMultimodalMessage
	System    string        `json:"system,omitempty"`
	MaxTokens int           `json:"max_tokens"`
	Stream    bool          `json:"stream"`
}

// ClaudeResponse は通常レスポンス
type ClaudeResponse struct {
	Content []ClaudeContent `json:"content"`
}

// ChatWithTools は Provider interface の実装（context対応）
func (p *ClaudeProvider) ChatWithTools(ctx context.Context, systemPrompt string, history []Message, model string) (string, error) {
	// モデル名はそのまま使用（ハードコードしない）
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}

	// Claudeのメッセージ構造に変換
	var messages []ClaudeMessage
	for _, msg := range history {
		messages = append(messages, ClaudeMessage(msg))
	}

	reqBody := ClaudeRequest{
		Model:     model,
		Messages:  messages,
		System:    systemPrompt,
		MaxTokens: 4096,
		Stream:    true,
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
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	// スピナー開始
	spinner := ui.NewSpinner()
	spinner.Start("Thinking")

	// 再利用可能なHTTPクライアントを使用
	resp, err := p.httpClient.Do(req)
	if err != nil {
		spinner.Stop()
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		spinner.Stop()

		// レート制限チェック
		if rateLimitErr := handleRateLimit(resp); rateLimitErr != nil {
			return "", rateLimitErr
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("API error (%d): unable to read response", resp.StatusCode)
		}
		return "", sanitizeErrorMessage(body, resp.StatusCode)
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
func (p *ClaudeProvider) handleStreamingResponse(ctx context.Context, resp *http.Response, spinner *ui.Spinner) (string, error) {
	// Claude固有のパース処理
	parser := func(line string) (string, bool, error) {
		if !strings.HasPrefix(line, "data: ") {
			return "", false, nil
		}

		data := strings.TrimPrefix(line, "data: ")
		var event ClaudeStreamEvent
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			return "", false, err
		}

		// message_stop イベントで終了
		if event.Type == "message_stop" {
			return "", true, nil
		}

		// content_block_delta イベントのみ処理
		if event.Type == "content_block_delta" && event.Delta.Type == "text_delta" {
			return event.Delta.Text, false, nil
		}

		return "", false, nil
	}

	return ParseStreamingResponse(ctx, resp, spinner, parser)
}

// handleNonStreamingResponse は非ストリーミングレスポンスを処理（フォールバック）
func (p *ClaudeProvider) handleNonStreamingResponse(resp *http.Response, spinner *ui.Spinner) (string, error) {
	var result ClaudeResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		spinner.Stop()
		return "", err
	}

	spinner.Stop()

	if len(result.Content) == 0 {
		return "", fmt.Errorf("no response from API")
	}

	content := result.Content[0].Text
	fmt.Println(content)
	return content, nil
}

// ChatWithImage は画像付きメッセージで会話を行う
func (p *ClaudeProvider) ChatWithImage(ctx context.Context, systemPrompt string, history []Message, userMessage string, image *ImageData, model string) (string, error) {
	// 画像がない場合は通常のChatWithToolsを使用
	if image == nil || image.Base64 == "" {
		history = append(history, Message{Role: "user", Content: userMessage})
		return p.ChatWithTools(ctx, systemPrompt, history, model)
	}

	// モデル名の設定
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}

	// 履歴をメッセージ配列に変換（テキストのみ）
	var messages []interface{}
	for _, msg := range history {
		messages = append(messages, ClaudeMessage(msg))
	}

	// 画像付きユーザーメッセージを追加
	multimodalMessage := ClaudeMultimodalMessage{
		Role: "user",
		Content: []ClaudeContentPart{
			{
				Type: "image",
				Source: &ClaudeImageSource{
					Type:      "base64",
					MediaType: image.MediaType,
					Data:      image.Base64,
				},
			},
			{
				Type: "text",
				Text: userMessage,
			},
		},
	}
	messages = append(messages, multimodalMessage)

	reqBody := ClaudeMultimodalRequest{
		Model:     model,
		Messages:  messages,
		System:    systemPrompt,
		MaxTokens: 4096,
		Stream:    true,
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
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	// スピナー開始
	spinner := ui.NewSpinner()
	spinner.Start("Analyzing image")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		spinner.Stop()
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		spinner.Stop()

		if rateLimitErr := handleRateLimit(resp); rateLimitErr != nil {
			return "", rateLimitErr
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("API error (%d): unable to read response", resp.StatusCode)
		}
		return "", sanitizeErrorMessage(body, resp.StatusCode)
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
