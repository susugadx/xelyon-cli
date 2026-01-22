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

// ClaudeCacheControl enables prompt caching for a content block.
//
// This is gated by config.PromptCache.Enabled and disabled by default.
// If the upstream schema changes, requests may fail; keep the feature optional.
type ClaudeCacheControl struct {
	Type string `json:"type"` // e.g. "ephemeral"
}

// ClaudeSystemBlock represents a system prompt content block.
//
// When prompt caching is enabled, we send system as an array of blocks instead of a string.
type ClaudeSystemBlock struct {
	Type         string              `json:"type"` // "text"
	Text         string              `json:"text"`
	CacheControl *ClaudeCacheControl `json:"cache_control,omitempty"`
}

// ClaudeThinkingConfig は Extended Thinking の設定
type ClaudeThinkingConfig struct {
	Type         string `json:"type"`          // "enabled"
	BudgetTokens int    `json:"budget_tokens"` // min 1024
}

type ClaudeRequest struct {
	Model    string          `json:"model"`
	Messages []ClaudeMessage `json:"messages"`
	// System can be either string (legacy) or []ClaudeSystemBlock (prompt caching).
	System    interface{}           `json:"system,omitempty"`
	MaxTokens int                   `json:"max_tokens"`
	Stream    bool                  `json:"stream"`
	Thinking  *ClaudeThinkingConfig `json:"thinking,omitempty"`
}

// buildClaudeSystemField builds the request "system" field.
//
// When prompt caching is enabled in config, it converts the system prompt into a single
// text block with cache_control to let Anthropic cache the prefix.
func buildClaudeSystemField(systemPrompt string) interface{} {
	cfg := config.GetGlobalConfig()
	if cfg == nil {
		return systemPrompt
	}
	if !cfg.PromptCache.Enabled {
		return systemPrompt
	}

	// NOTE: The cache_control schema is based on Anthropic prompt caching examples.
	// We keep it minimal here.
	return []ClaudeSystemBlock{
		{
			Type: "text",
			Text: systemPrompt,
			CacheControl: &ClaudeCacheControl{
				Type: "ephemeral",
			},
		},
	}
}

// levelToBudgetTokens は Thinking Level を budget_tokens に変換
func levelToBudgetTokens(level string) int {
	switch level {
	case "low":
		return 5000
	case "medium":
		return 10000
	case "high":
		return 20000
	case "xhigh":
		return 40000
	default:
		return 10000
	}
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
	Model    string        `json:"model"`
	Messages []interface{} `json:"messages"` // ClaudeMessage or ClaudeMultimodalMessage
	// System can be either string (legacy) or []ClaudeSystemBlock (prompt caching).
	System    interface{}           `json:"system,omitempty"`
	MaxTokens int                   `json:"max_tokens"`
	Stream    bool                  `json:"stream"`
	Thinking  *ClaudeThinkingConfig `json:"thinking,omitempty"`
}

// ClaudeResponse は通常レスポンス
type ClaudeResponse struct {
	Content []ClaudeContent `json:"content"`
}

// claudeRequestResult はexecuteClaudeRequestの結果を格納
type claudeRequestResult struct {
	Response *http.Response
	Spinner  *ui.Spinner
}

// executeClaudeRequest はClaude API呼び出しの共通処理
// withImage: 画像付きリクエストの場合はtrue（スピナー表示に影響）
func (p *ClaudeProvider) executeClaudeRequest(ctx context.Context, reqBody interface{}, withImage bool) (*claudeRequestResult, error) {
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	spinner := StartThinkingSpinner(withImage, "")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		spinner.Stop()
		return nil, err
	}

	if resp.StatusCode != 200 {
		spinner.Stop()
		defer resp.Body.Close()

		if rateLimitErr := handleRateLimit(resp); rateLimitErr != nil {
			return nil, rateLimitErr
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("API error (%d): unable to read response", resp.StatusCode)
		}
		return nil, sanitizeErrorMessage(body, resp.StatusCode)
	}

	return &claudeRequestResult{Response: resp, Spinner: spinner}, nil
}

// processClaudeResponse はレスポンス処理（ストリーミング/非ストリーミング）
func (p *ClaudeProvider) processClaudeResponse(ctx context.Context, result *claudeRequestResult) (string, error) {
	defer result.Response.Body.Close()

	contentType := result.Response.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/event-stream") {
		return p.handleStreamingResponse(ctx, result.Response, result.Spinner)
	}
	return p.handleNonStreamingResponse(result.Response, result.Spinner)
}

// ChatWithTools は Provider interface の実装（context対応）
func (p *ClaudeProvider) ChatWithTools(ctx context.Context, systemPrompt string, history []Message, model string) (string, error) {
	// モデル名を設定（config優先、フォールバックはclaude-sonnet-4-20250514）
	model = GetDefaultModel(model, "claude", "claude-sonnet-4-20250514")

	// Claudeのメッセージ構造に変換
	var messages []ClaudeMessage
	for _, msg := range history {
		messages = append(messages, ClaudeMessage(msg))
	}

	cfg := config.GetGlobalConfig()

	reqBody := ClaudeRequest{
		Model:     model,
		Messages:  messages,
		System:    buildClaudeSystemField(systemPrompt),
		MaxTokens: 4096,
		Stream:    true,
	}

	// Extended Thinking 適用
	if cfg.Thinking.Enabled {
		reqBody.Thinking = &ClaudeThinkingConfig{
			Type:         "enabled",
			BudgetTokens: levelToBudgetTokens(cfg.Thinking.Level),
		}
	}

	result, err := p.executeClaudeRequest(ctx, reqBody, false)
	if err != nil {
		return "", err
	}

	return p.processClaudeResponse(ctx, result)
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

	// モデル名を設定（config優先、フォールバックはclaude-sonnet-4-20250514）
	model = GetDefaultModel(model, "claude", "claude-sonnet-4-20250514")

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

	cfg := config.GetGlobalConfig()

	reqBody := ClaudeMultimodalRequest{
		Model:     model,
		Messages:  messages,
		System:    buildClaudeSystemField(systemPrompt),
		MaxTokens: 4096,
		Stream:    true,
	}

	// Extended Thinking 適用
	if cfg.Thinking.Enabled {
		reqBody.Thinking = &ClaudeThinkingConfig{
			Type:         "enabled",
			BudgetTokens: levelToBudgetTokens(cfg.Thinking.Level),
		}
	}

	result, err := p.executeClaudeRequest(ctx, reqBody, true)
	if err != nil {
		return "", err
	}

	return p.processClaudeResponse(ctx, result)
}
