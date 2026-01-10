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

const defaultOpenAIURL = "https://api.openai.com/v1/chat/completions"

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
	Model    string        `json:"model"`
	Messages []interface{} `json:"messages"` // Message or OpenAIMultimodalMessage
	Stream   bool          `json:"stream"`
}

// ChatWithTools は Provider interface の実装（context対応）
func (p *OpenAIProvider) ChatWithTools(ctx context.Context, systemPrompt string, history []Message, model string) (string, error) {
	// メッセージ構築
	messages := []Message{
		{Role: "system", Content: systemPrompt},
	}
	messages = append(messages, history...)

	// モデル名はそのまま使用（ハードコードしない）
	if model == "" {
		model = "gpt-4o"
	}

	reqBody := ChatRequest{
		Model:    model,
		Messages: messages,
		Stream:   true,
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
	var result ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		spinner.Stop()
		return "", err
	}

	spinner.Stop()

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no response from API")
	}

	content := result.Choices[0].Message.Content
	fmt.Println(content)
	return content, nil
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
