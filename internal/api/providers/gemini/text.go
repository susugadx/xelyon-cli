package gemini

import (
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

// chatWithTextMode はテキストベースのツール呼び出しモード（従来の実装）
func (p *Provider) chatWithTextMode(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	// モデル名を設定（config優先、フォールバックはgemini-3.1-pro-preview-customtools）
	model = api.GetDefaultModel(model, "gemini", "gemini-3.1-pro-preview-customtools")

	// キャッシュ管理（テキストモードではツール定義なし）
	cacheName, msgsToSend, err := p.updateOrUseCache(ctx, systemPrompt, history, model, nil, nil)
	if err != nil {
		return "", err
	}

	// System prompt を system_instruction フィールドに設定（キャッシュ利用時は不要）
	var sysInstruction *GeminiSystemInstruction
	if cacheName == "" && systemPrompt != "" {
		sysInstruction = &GeminiSystemInstruction{
			Parts: []GeminiPart{{Text: systemPrompt}},
		}
	}

	// Geminiのメッセージ構造に変換
	var contents []GeminiContent

	// 会話履歴を変換（msgsToSendを使用）
	for _, msg := range msgsToSend {
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
		CachedContent:     cacheName,
		SystemInstruction: sysInstruction,
		Contents:          contents,
	}

	// Thinking 設定（Gemini 3 vs 2.5 で自動分岐）
	reqBody.GenerationConfig = getThinkingConfigForModel(ctx, model, cfg)

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	// Gemini API endpoint（ストリーミング）
	url := getGeminiURL(model)

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
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
		if isFlash && !api.IsThinkingEnabled(ctx) {
			msg = "Thinking"
		} else {
			msg = "Deep thinking"
		}
		spinner = ui.NewSpinner()
		spinner.Start(msg)
		ui.SetGlobalSpinner(spinner)
	} else {
		spinner = api.StartThinkingSpinner(ctx, false, "")
	}

	// 503 リトライ付き HTTP リクエスト
	resp, err := p.doRequestWithRetry(ctx, req, jsonBody)
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
	// streamGenerateContent?alt=sse を使用しているため、常に SSE パーサーを使用する
	return p.handleSSEResponse(ctx, resp, spinner)
}

// ChatWithImage は画像付きメッセージで会話を行う
func (p *Provider) ChatWithImage(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
	// 画像がない場合は通常のChatWithToolsを使用
	if image == nil || image.Base64 == "" {
		history = append(history, api.Message{Role: "user", Content: userMessage})
		return p.ChatWithTools(ctx, systemPrompt, history, model)
	}

	// モデル名を設定（config優先、フォールバックはgemini-3.1-pro-preview-customtools）
	model = api.GetDefaultModel(model, "gemini", "gemini-3.1-pro-preview-customtools")

	// System prompt を system_instruction フィールドに設定
	var sysInstruction *GeminiSystemInstruction
	if systemPrompt != "" {
		sysInstruction = &GeminiSystemInstruction{
			Parts: []GeminiPart{{Text: systemPrompt}},
		}
	}

	// contentsを構築
	var contents []interface{}

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
		SystemInstruction: sysInstruction,
		Contents:          contents,
	}

	// FC有効時はTools/ToolConfigを追加（画像+FC対応）
	if p.IsFunctionCallingEnabled() {
		reqBody.Tools = GetCombinedToolDefinitions(p.mcpTools)
		fcMode := os.Getenv("GEMINI_FC_MODE")
		if fcMode == "" {
			fcMode = "AUTO"
		}
		reqBody.ToolConfig = &GeminiToolConfigWrapper{
			FunctionCallingConfig: GeminiFunctionCallingConfig{Mode: fcMode},
		}
	}

	// Thinking 設定（Gemini 3 vs 2.5 で自動分岐）
	reqBody.GenerationConfig = getThinkingConfigForModel(ctx, model, cfgImg)

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	// URL: 常に SSE 対応エンドポイントを使用
	url := getGeminiURL(model)

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
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
		if isFlash && !api.IsThinkingEnabled(ctx) {
			msg = "Analyzing image"
		} else {
			msg = "Deep thinking (image)"
		}
		spinner = ui.NewSpinner()
		spinner.Start(msg)
		ui.SetGlobalSpinner(spinner)
	} else {
		spinner = api.StartThinkingSpinner(ctx, true, "") // isImage=true
	}

	// 503 リトライ付き HTTP リクエスト
	resp, err := p.doRequestWithRetry(ctx, req, jsonBody)
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

	// 常にストリーミング処理（SSE）
	return p.handleSSEResponse(ctx, resp, spinner)
}
