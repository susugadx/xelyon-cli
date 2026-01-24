package claude

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func init() {
	api.RegisterProvider("claude", func(apiKey string) (api.Provider, error) {
		if apiKey == "" {
			return nil, fmt.Errorf("ANTHROPIC_API_KEY not set")
		}
		return New(apiKey), nil
	})
	// anthropic エイリアス
	api.RegisterProvider("anthropic", func(apiKey string) (api.Provider, error) {
		if apiKey == "" {
			return nil, fmt.Errorf("ANTHROPIC_API_KEY not set")
		}
		return New(apiKey), nil
	})
}

const defaultClaudeURL = "https://api.anthropic.com/v1/messages"

// Provider はClaude (Anthropic) APIのプロバイダー実装
type Provider struct {
	apiKey     string
	apiURL     string
	httpClient *http.Client
}

// New は新しいProviderを作成
func New(apiKey string) *Provider {
	// 環境変数からURLをオーバーライド可能
	apiURL := os.Getenv("ANTHROPIC_API_URL")
	if apiURL == "" {
		apiURL = defaultClaudeURL
	}

	return &Provider{
		apiKey: apiKey,
		apiURL: apiURL,
		httpClient: &http.Client{
			Timeout: config.DefaultHTTPTimeout,
		},
	}
}

// Name はプロバイダー名を返す
func (p *Provider) Name() string {
	return "Claude"
}

// SupportsImages は画像入力対応を返す
func (p *Provider) SupportsImages() bool {
	return true
}

// Message はClaudeのメッセージ構造
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ClaudeRequest はClaude APIリクエスト

// CacheControl enables prompt caching for a content block.
//
// This is gated by config.PromptCache.Enabled and disabled by default.
// If the upstream schema changes, requests may fail; keep the feature optional.
type CacheControl struct {
	Type string `json:"type"` // e.g. "ephemeral"
}

// SystemBlock represents a system prompt content block.
//
// When prompt caching is enabled, we send system as an array of blocks instead of a string.
type SystemBlock struct {
	Type         string        `json:"type"` // "text"
	Text         string        `json:"text"`
	CacheControl *CacheControl `json:"cache_control,omitempty"`
}

// ThinkingConfig は Extended Thinking の設定
type ThinkingConfig struct {
	Type         string `json:"type"`          // "enabled"
	BudgetTokens int    `json:"budget_tokens"` // min 1024
}

type Request struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	// System can be either string (legacy) or []SystemBlock (prompt caching).
	System    interface{}     `json:"system,omitempty"`
	MaxTokens int             `json:"max_tokens"`
	Stream    bool            `json:"stream"`
	Thinking  *ThinkingConfig `json:"thinking,omitempty"`
}

// buildSystemField builds the request "system" field.
//
// When prompt caching is enabled in config, it converts the system prompt into a single
// text block with cache_control to let Anthropic cache the prefix.
func buildSystemField(systemPrompt string) interface{} {
	cfg := config.GetGlobalConfig()
	if cfg == nil {
		return systemPrompt
	}
	if !cfg.PromptCache.Enabled {
		return systemPrompt
	}

	// NOTE: The cache_control schema is based on Anthropic prompt caching examples.
	// We keep it minimal here.
	return []SystemBlock{
		{
			Type: "text",
			Text: systemPrompt,
			CacheControl: &CacheControl{
				Type: "ephemeral",
			},
		},
	}
}

// LevelToBudgetTokens は api.LevelToBudgetTokens のエイリアス（後方互換）
func LevelToBudgetTokens(level string) int {
	return api.LevelToBudgetTokens(level)
}

// Delta はストリームの差分
type Delta struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// StreamEvent はストリームイベント
type StreamEvent struct {
	Type  string `json:"type"`
	Delta Delta  `json:"delta"`
}

// Content はレスポンスのコンテンツ
type Content struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// ContentPart はマルチモーダルコンテンツのパート
type ContentPart struct {
	Type   string       `json:"type"`             // "text" or "image"
	Text   string       `json:"text,omitempty"`   // type="text"の場合
	Source *ImageSource `json:"source,omitempty"` // type="image"の場合
}

// ImageSource は画像ソース
type ImageSource struct {
	Type      string `json:"type"`       // "base64"
	MediaType string `json:"media_type"` // "image/png", "image/jpeg" etc
	Data      string `json:"data"`       // Base64エンコードされたデータ
}

// MultimodalMessage はマルチモーダルメッセージ（画像含む）
type MultimodalMessage struct {
	Role    string        `json:"role"`
	Content []ContentPart `json:"content"`
}

// MultimodalRequest はマルチモーダルAPIリクエスト
type MultimodalRequest struct {
	Model    string        `json:"model"`
	Messages []interface{} `json:"messages"` // Message or MultimodalMessage
	// System can be either string (legacy) or []SystemBlock (prompt caching).
	System    interface{}     `json:"system,omitempty"`
	MaxTokens int             `json:"max_tokens"`
	Stream    bool            `json:"stream"`
	Thinking  *ThinkingConfig `json:"thinking,omitempty"`
}

// Response は通常レスポンス
type Response struct {
	Content []Content `json:"content"`
}

// requestResult はexecuteRequestの結果を格納
type requestResult struct {
	Response *http.Response
	Spinner  *ui.Spinner
}

// executeRequest はClaude API呼び出しの共通処理
// withImage: 画像付きリクエストの場合はtrue（スピナー表示に影響）
func (p *Provider) executeRequest(ctx context.Context, reqBody interface{}, withImage bool) (*requestResult, error) {
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

	spinner := api.StartThinkingSpinner(withImage, "")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		spinner.Stop()
		return nil, err
	}

	if resp.StatusCode != 200 {
		spinner.Stop()
		defer resp.Body.Close()

		if rateLimitErr := api.HandleRateLimit(resp); rateLimitErr != nil {
			return nil, rateLimitErr
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("API error (%d): unable to read response", resp.StatusCode)
		}
		return nil, api.SanitizeErrorMessage(body, resp.StatusCode)
	}

	return &requestResult{Response: resp, Spinner: spinner}, nil
}

// processResponse はレスポンス処理（ストリーミング/非ストリーミング）
func (p *Provider) processResponse(ctx context.Context, result *requestResult) (string, error) {
	defer result.Response.Body.Close()

	contentType := result.Response.Header.Get("Content-Type")
	if strings.Contains(contentType, "text/event-stream") {
		return p.handleStreamingResponse(ctx, result.Response, result.Spinner)
	}
	return p.handleNonStreamingResponse(result.Response, result.Spinner)
}

// ChatWithTools は Provider interface の実装（context対応）
func (p *Provider) ChatWithTools(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	// モデル名を設定（config優先、フォールバックはclaude-sonnet-4-20250514）
	model = api.GetDefaultModel(model, "claude", "claude-sonnet-4-20250514")

	// Claudeのメッセージ構造に変換
	var messages []Message
	for _, msg := range history {
		messages = append(messages, Message(msg))
	}

	cfg := config.GetGlobalConfig()

	reqBody := Request{
		Model:     model,
		Messages:  messages,
		System:    buildSystemField(systemPrompt),
		MaxTokens: 4096,
		Stream:    true,
	}

	// Extended Thinking 適用
	if cfg.Thinking.Enabled {
		reqBody.Thinking = &ThinkingConfig{
			Type:         "enabled",
			BudgetTokens: LevelToBudgetTokens(cfg.Thinking.Level),
		}
	}

	result, err := p.executeRequest(ctx, reqBody, false)
	if err != nil {
		return "", err
	}

	return p.processResponse(ctx, result)
}

// handleStreamingResponse はストリーミングレスポンスを処理
func (p *Provider) handleStreamingResponse(ctx context.Context, resp *http.Response, spinner *ui.Spinner) (string, error) {
	// Claude固有のパース処理
	parser := func(line string) (string, bool, error) {
		if !strings.HasPrefix(line, "data: ") {
			return "", false, nil
		}

		data := strings.TrimPrefix(line, "data: ")
		var event StreamEvent
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

	return api.ParseStreamingResponse(ctx, resp, spinner, parser)
}

// handleNonStreamingResponse は非ストリーミングレスポンスを処理（フォールバック）
func (p *Provider) handleNonStreamingResponse(resp *http.Response, spinner *ui.Spinner) (string, error) {
	var result Response
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
func (p *Provider) ChatWithImage(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
	// 画像がない場合は通常のChatWithToolsを使用
	if image == nil || image.Base64 == "" {
		history = append(history, api.Message{Role: "user", Content: userMessage})
		return p.ChatWithTools(ctx, systemPrompt, history, model)
	}

	// モデル名を設定（config優先、フォールバックはclaude-sonnet-4-20250514）
	model = api.GetDefaultModel(model, "claude", "claude-sonnet-4-20250514")

	// 履歴をメッセージ配列に変換（テキストのみ）
	var messages []interface{}
	for _, msg := range history {
		messages = append(messages, Message(msg))
	}

	// 画像付きユーザーメッセージを追加
	multimodalMessage := MultimodalMessage{
		Role: "user",
		Content: []ContentPart{
			{
				Type: "image",
				Source: &ImageSource{
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

	reqBody := MultimodalRequest{
		Model:     model,
		Messages:  messages,
		System:    buildSystemField(systemPrompt),
		MaxTokens: 4096,
		Stream:    true,
	}

	// Extended Thinking 適用
	if cfg.Thinking.Enabled {
		reqBody.Thinking = &ThinkingConfig{
			Type:         "enabled",
			BudgetTokens: LevelToBudgetTokens(cfg.Thinking.Level),
		}
	}

	result, err := p.executeRequest(ctx, reqBody, true)
	if err != nil {
		return "", err
	}

	return p.processResponse(ctx, result)
}
