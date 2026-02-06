package gemini

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// chatWithTextMode はテキストベースのツール呼び出しモード（従来の実装）
func (p *Provider) chatWithTextMode(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	// モデル名を設定（config優先、フォールバックはgemini-2.0-flash）
	model = api.GetDefaultModel(model, "gemini", "gemini-2.0-flash")

	// Geminiのメッセージ構造に変換
	var contents []GeminiContent

	// System promptを最初のユーザーメッセージとして追加
	if systemPrompt != "" {
		contents = append(contents, GeminiContent{
			Parts: []GeminiPart{{Text: systemPrompt}},
			Role:  "user",
		})
		// ダミーのモデル応答を追加（Geminiは交互のroleを期待）
		contents = append(contents, GeminiContent{
			Parts: []GeminiPart{{Text: "Understood. I'll follow these instructions."}},
			Role:  "model",
		})
	}

	// 会話履歴を変換
	for _, msg := range history {
		role := "user"
		if msg.Role == "assistant" {
			role = "model"
		}
		contents = append(contents, GeminiContent{
			Parts: []GeminiPart{{Text: msg.Content}},
			Role:  role,
		})
	}

	cfg := config.GetGlobalConfig()

	reqBody := GeminiRequest{
		Contents: contents,
	}

	// Thinking 設定（Gemini 3 vs 2.5 で自動分岐）
	reqBody.GenerationConfig = getThinkingConfigForModel(model, cfg)

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	// Gemini API endpoint（ストリーミング）
	url := getGeminiURL(model)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", p.apiKey) // APIキーはヘッダーで送信（セキュリティ向上）

	// スピナー開始
	// Gemini 3 Flash (minimal) は "Thinking"、Pro または thinking.enabled=true は "Deep thinking"
	var spinner *ui.Spinner
	if isGemini3Model(model) {
		isFlash := strings.Contains(model, "flash")
		var msg string
		if isFlash && !cfg.Thinking.Enabled {
			msg = "Thinking"
		} else {
			msg = "Deep thinking"
		}
		spinner = ui.NewSpinner()
		spinner.Start(msg)
		ui.SetGlobalSpinner(spinner)
	} else {
		spinner = api.StartThinkingSpinner(false, "")
	}

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
		if rateLimitErr := api.HandleRateLimit(resp); rateLimitErr != nil {
			return "", rateLimitErr
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("gemini API error (status %d): unable to read response body - %v", resp.StatusCode, err)
		}
		if len(body) == 0 {
			return "", fmt.Errorf("gemini API error (status %d): empty response body. Check API key and model name '%s'", resp.StatusCode, model)
		}
		return "", api.SanitizeErrorMessage(body, resp.StatusCode)
	}

	// Content-Typeでストリーミング対応を判定
	contentType := resp.Header.Get("Content-Type")
	isStreaming := contentType == "" || // デフォルトはストリーミング
		len(contentType) > 0 // Content-Typeがあればストリーミングとして処理

	if isStreaming {
		return p.handleStreamingResponse(ctx, resp, spinner)
	} else {
		return p.handleNonStreamingResponse(resp, spinner)
	}
}

// ChatWithImage は画像付きメッセージで会話を行う
func (p *Provider) ChatWithImage(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
	// 画像がない場合は通常のChatWithToolsを使用
	if image == nil || image.Base64 == "" {
		history = append(history, api.Message{Role: "user", Content: userMessage})
		return p.ChatWithTools(ctx, systemPrompt, history, model)
	}

	// モデル名を設定（config優先、フォールバックはgemini-2.0-flash）
	model = api.GetDefaultModel(model, "gemini", "gemini-2.0-flash")

	// contentsを構築
	var contents []interface{}

	// System promptを最初のユーザーメッセージとして追加
	if systemPrompt != "" {
		contents = append(contents, GeminiContent{
			Parts: []GeminiPart{{Text: systemPrompt}},
			Role:  "user",
		})
		// ダミーのモデル応答を追加（Geminiは交互のroleを期待）
		contents = append(contents, GeminiContent{
			Parts: []GeminiPart{{Text: "Understood. I'll follow these instructions."}},
			Role:  "model",
		})
	}

	// 会話履歴を変換（テキストのみ）
	for _, msg := range history {
		role := "user"
		if msg.Role == "assistant" {
			role = "model"
		}
		contents = append(contents, GeminiContent{
			Parts: []GeminiPart{{Text: msg.Content}},
			Role:  role,
		})
	}

	// 画像付きユーザーメッセージを追加
	multimodalContent := GeminiMultimodalContent{
		Role: "user",
		Parts: []GeminiMultimodalPart{
			{
				InlineData: &GeminiInlineData{
					MimeType: image.MediaType,
					Data:     image.Base64,
				},
			},
			{
				Text: userMessage,
			},
		},
	}
	contents = append(contents, multimodalContent)

	cfgImg := config.GetGlobalConfig()

	reqBody := GeminiMultimodalRequest{
		Contents: contents,
	}

	// Thinking 設定（Gemini 3 vs 2.5 で自動分岐）
	reqBody.GenerationConfig = getThinkingConfigForModel(model, cfgImg)

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	// Gemini API endpoint（ストリーミング）
	url := getGeminiURL(model)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", p.apiKey)

	// スピナー開始（画像モード）
	// Gemini 3 Flash (minimal) は "Analyzing image"、Pro または thinking.enabled=true は "Deep thinking (image)"
	var spinner *ui.Spinner
	if isGemini3Model(model) {
		isFlash := strings.Contains(model, "flash")
		var msg string
		if isFlash && !cfgImg.Thinking.Enabled {
			msg = "Analyzing image"
		} else {
			msg = "Deep thinking (image)"
		}
		spinner = ui.NewSpinner()
		spinner.Start(msg)
		ui.SetGlobalSpinner(spinner)
	} else {
		spinner = api.StartThinkingSpinner(true, "") // isImage=true
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		spinner.Stop()
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		spinner.Stop()

		if rateLimitErr := api.HandleRateLimit(resp); rateLimitErr != nil {
			return "", rateLimitErr
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("gemini API error (status %d): unable to read response body - %v", resp.StatusCode, err)
		}
		if len(body) == 0 {
			return "", fmt.Errorf("gemini API error (status %d): empty response body. Check API key and model name '%s'", resp.StatusCode, model)
		}
		return "", api.SanitizeErrorMessage(body, resp.StatusCode)
	}

	// ストリーミング処理
	contentType := resp.Header.Get("Content-Type")
	isStreaming := contentType == "" || len(contentType) > 0

	if isStreaming {
		return p.handleStreamingResponse(ctx, resp, spinner)
	} else {
		return p.handleNonStreamingResponse(resp, spinner)
	}
}
