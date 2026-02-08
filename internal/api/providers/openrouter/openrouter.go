package openrouter

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/api/providers/openai"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func init() {
	api.RegisterProvider("openrouter", func(apiKey string) (api.Provider, error) {
		if apiKey == "" {
			return nil, fmt.Errorf("OPENROUTER_API_KEY not set")
		}
		return New(apiKey), nil
	})
}

var yellow = color.New(color.FgYellow)

const defaultOpenRouterURL = "https://openrouter.ai/api/v1/chat/completions"

// toolCallAccumulator はストリーミング中のtool_callを蓄積する
type toolCallAccumulator struct {
	ID        string
	Name      string
	Arguments strings.Builder
}

// ContentPart はマルチモーダルコンテンツのパート（OpenAI互換）
type ContentPart struct {
	Type     string    `json:"type"`                // "text" or "image_url"
	Text     string    `json:"text,omitempty"`      // type="text"の場合
	ImageURL *ImageURL `json:"image_url,omitempty"` // type="image_url"の場合
}

// ImageURL は画像URL
type ImageURL struct {
	URL string `json:"url"` // "data:image/png;base64,..." 形式
}

// MultimodalMessage はマルチモーダルメッセージ（OpenAI互換）
type MultimodalMessage struct {
	Role    string        `json:"role"`
	Content []ContentPart `json:"content"`
}

// Provider はOpenRouter APIのプロバイダー実装（OpenAI互換）
type Provider struct {
	api.BaseProvider
	mcpTools      []api.ToolDefinition // MCP ツール定義（Function Calling用）
	usageCallback api.UsageCallback    // トークン使用量コールバック
}

// New は新しいProviderを作成
func New(apiKey string) *Provider {
	return &Provider{
		BaseProvider: api.NewBaseProvider("OpenRouter", apiKey, defaultOpenRouterURL, "OPENROUTER_API_URL"),
	}
}

// SupportsImages は画像入力対応を返す
// OpenRouterは複数モデルに対応するため、モデルによっては画像対応
func (p *Provider) SupportsImages() bool {
	return true
}

// IsFunctionCallingEnabled は Function Calling が有効かを返す
// OPENROUTER_FUNCTION_CALLING=0 で無効化可能
func (p *Provider) IsFunctionCallingEnabled() bool {
	return os.Getenv("OPENROUTER_FUNCTION_CALLING") != "0"
}

// ChatWithTools は Provider interface の実装（context対応）
func (p *Provider) ChatWithTools(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	// Extended Thinking 非対応警告
	cfg := config.GetGlobalConfig()
	if cfg.Thinking.Enabled {
		yellow.Println("⚠️  Warning: OpenRouter does not support Extended Thinking. Proceeding without it.")
	}

	// メッセージ構築
	messages := []api.Message{
		{Role: "system", Content: systemPrompt},
	}
	messages = append(messages, history...)

	// モデル名を設定（config優先、フォールバックはanthropic/claude-opus-4.5）
	model = api.GetDefaultModel(model, "openrouter", "anthropic/claude-opus-4.5")

	reqBody := api.ChatRequest{
		Model:         model,
		Messages:      messages,
		MaxTokens:     api.GetMaxOutputTokens("openrouter", model),
		Stream:        true,
		StreamOptions: &api.StreamOptions{IncludeUsage: true},
	}

	// Function Calling: ツール定義を追加（環境変数で無効化可能）
	if os.Getenv("OPENROUTER_FUNCTION_CALLING") != "0" {
		reqBody.Tools = openai.GetCombinedOpenAITools(p.mcpTools)
		reqBody.ToolChoice = "auto"
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.APIURL(), bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.APIKey)
	// OpenRouter 固有ヘッダー（オプション）
	req.Header.Set("HTTP-Referer", "https://github.com/susugadx/xelyon-cli")
	req.Header.Set("X-Title", "XELYON CLI")

	// スピナー開始
	spinner := api.StartThinkingSpinner(false, "")

	// 再利用可能なHTTPクライアントを使用
	resp, err := p.HTTPClient.Do(req)
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
	var fullResponse strings.Builder
	var toolCallsOutput strings.Builder
	toolCalls := make(map[int]*toolCallAccumulator)
	scanner := bufio.NewScanner(resp.Body)
	firstChunk := true
	var lastUsage *api.Usage

	for scanner.Scan() {
		// contextキャンセルチェック
		select {
		case <-ctx.Done():
			spinner.Stop()
			return "", ctx.Err()
		default:
		}

		line := scanner.Text()
		if strings.HasPrefix(line, "data: ") {
			data := strings.TrimPrefix(line, "data: ")
			if data == "[DONE]" {
				break
			}

			var streamResp api.StreamResponse
			if err := json.Unmarshal([]byte(data), &streamResp); err != nil {
				// JSONパースエラーを警告（データ損失を防ぐため記録）
				fmt.Fprintf(os.Stderr, "⚠️  Warning: failed to parse streaming response: %v\n", err)
				continue
			}

			// Usage情報を追跡（最終チャンクに含まれる）
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
			}

			if len(streamResp.Choices) > 0 {
				choice := streamResp.Choices[0]

				// Function Calling: tool_calls を蓄積
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

				// Function Calling: finish_reason == "tool_calls" で完了
				if choice.FinishReason == "tool_calls" {
					// tool_calls を内部JSON形式に変換
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
						if toolJSON, err := openai.ConvertToolCallToToolJSON(tc); err == nil {
							toolCallsOutput.WriteString(toolJSON)
						}
					}
				}

				content := choice.Delta.Content

				// 最初のコンテンツでスピナー停止 + AI発言ヘッダー表示
				if firstChunk && content != "" {
					spinner.Stop()
					firstChunk = false
					api.PrintAIHeader()
				}

				fmt.Print(content)
				fullResponse.WriteString(content)
			}
		}
	}

	// スキャナーのI/Oエラーチェック
	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("stream reading error: %w", err)
	}

	// Usage callback
	if p.usageCallback != nil && lastUsage != nil {
		p.usageCallback(*lastUsage)
	}

	// tool_calls がある場合はそれを返す
	if toolCallsOutput.Len() > 0 {
		spinner.Stop()
		if fullResponse.Len() > 0 {
			fmt.Println()
			return fullResponse.String() + toolCallsOutput.String(), nil
		}
		return toolCallsOutput.String(), nil
	}

	fmt.Println()
	return fullResponse.String(), nil
}

// handleNonStreamingResponse は非ストリーミングレスポンスを処理（フォールバック）
func (p *Provider) handleNonStreamingResponse(resp *http.Response, spinner *ui.Spinner) (string, error) {
	return api.HandleNonStreamingResponse(resp, spinner)
}

// ChatWithImage は画像付きメッセージで会話を行う
func (p *Provider) ChatWithImage(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
	// 画像がない場合はテキストのみで送信
	if image == nil || image.Base64 == "" {
		history = append(history, api.Message{Role: "user", Content: userMessage})
		return p.ChatWithTools(ctx, systemPrompt, history, model)
	}

	// 画像付きメッセージの場合、専用のリクエスト構造を使用
	return p.chatWithImageRequest(ctx, systemPrompt, history, userMessage, image, model)
}

// chatWithImageRequest は画像付きメッセージ用のリクエストを送信
func (p *Provider) chatWithImageRequest(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
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

	// モデル名を設定
	model = api.GetDefaultModel(model, "openrouter", "anthropic/claude-opus-4.5")

	// リクエスト構造体
	type multimodalRequest struct {
		Model         string        `json:"model"`
		Messages      []interface{} `json:"messages"`
		MaxTokens     int           `json:"max_tokens,omitempty"`
		Stream        bool          `json:"stream"`
		StreamOptions interface{}   `json:"stream_options,omitempty"`
		Tools         interface{}   `json:"tools,omitempty"`
		ToolChoice    string        `json:"tool_choice,omitempty"`
	}

	reqBody := multimodalRequest{
		Model:         model,
		Messages:      messages,
		MaxTokens:     api.GetMaxOutputTokens("openrouter", model),
		Stream:        true,
		StreamOptions: &api.StreamOptions{IncludeUsage: true},
	}

	// Function Calling: ツール定義を追加（環境変数で無効化可能）
	if os.Getenv("OPENROUTER_FUNCTION_CALLING") != "0" {
		reqBody.Tools = openai.GetCombinedOpenAITools(p.mcpTools)
		reqBody.ToolChoice = "auto"
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.BaseProvider.APIURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.APIKey)
	req.Header.Set("HTTP-Referer", "https://github.com/susugadx/xelyon-cli")
	req.Header.Set("X-Title", "XELYON CLI")

	// スピナー開始
	spinner := api.StartThinkingSpinner(true, "")

	resp, err := p.HTTPClient.Do(req)
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
	}
	return p.handleNonStreamingResponse(resp, spinner)
}

// APIURL はテスト用にAPIURLを公開
func (p *Provider) APIURL() string {
	return p.BaseProvider.APIURL
}

// SetMCPTools は MCP ツール定義を設定する（Function Calling用）
func (p *Provider) SetMCPTools(tools []api.ToolDefinition) {
	p.mcpTools = tools
}

// SetUsageCallback は使用量レポートのコールバックを設定する
func (p *Provider) SetUsageCallback(callback api.UsageCallback) {
	p.usageCallback = callback
}
