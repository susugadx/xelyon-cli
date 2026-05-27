package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

// chatWithTextMode はテキストベースのツール呼び出しモード（従来の実装）
func (p *Provider) chatWithTextMode(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	// モデル名を設定（config優先、フォールバックはgemini-3.1-pro-preview-customtools）
	model = api.ResolveProviderRequestModel(ctx, model, "gemini")

	// キャッシュ管理（テキストモードではツール定義なし）
	cacheName, msgsToSend, err := p.updateOrUseCache(ctx, systemPrompt, history, model, nil, nil)
	if err != nil {
		return "", err
	}

	cfg := config.FromContext(ctx)
	reqBody := buildGeminiTextRequest(ctx, systemPrompt, msgsToSend, model, cacheName, cfg)

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

	// スピナー開始: レスポンス開始前は "Waiting for Gemini..." を表示
	// SSEストリーム開始後に thinking メッセージに切り替える
	thinkingMsg := getThinkingSpinnerMessage(ctx, model, false)
	spinner := api.StartSpinnerWithMessage(ctx, "Waiting for Gemini...")

	// 503 リトライ付き HTTP リクエスト
	resp, err := p.doRequestWithRetry(ctx, req, jsonBody, model)
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
	return p.handleSSEResponse(ctx, resp, spinner, thinkingMsg, model)
}

// ChatWithImage は画像付きメッセージで会話を行う
func (p *Provider) ChatWithImage(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
	// 画像がない場合は通常のChatWithToolsを使用
	if image == nil || image.Base64 == "" {
		history = append(history, api.Message{Role: "user", Content: userMessage})
		return p.ChatWithTools(ctx, systemPrompt, history, model)
	}

	// モデル名を設定（config優先、フォールバックはgemini-3.1-pro-preview-customtools）
	model = api.ResolveProviderRequestModel(ctx, model, "gemini")

	cfgImg := config.FromContext(ctx)
	functionCallingPolicy := newGeminiFunctionCallingPolicy(cfgImg, model)
	if !functionCallingPolicy.Enabled() && !api.IsToolUseDisabled(ctx) {
		return "", functionCallingPolicy.UnsupportedError()
	}
	reqBody := buildGeminiMultimodalRequest(ctx, systemPrompt, history, userMessage, image, model, p.mcpTools, functionCallingPolicy.Enabled(), cfgImg)

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

	// スピナー開始: レスポンス開始前は "Waiting for Gemini..." を表示
	// SSEストリーム開始後に thinking メッセージに切り替える
	thinkingMsg := getThinkingSpinnerMessage(ctx, model, true)
	spinner := api.StartSpinnerWithMessage(ctx, "Waiting for Gemini...")

	// 503 リトライ付き HTTP リクエスト
	resp, err := p.doRequestWithRetry(ctx, req, jsonBody, model)
	if err != nil {
		spinner.Stop()
		if retryResult, handled, retryErr := retryGeminiTimeout(ctx, err, "multimodal FC mode", func(retryCtx context.Context) (string, error) {
			return p.ChatWithImage(retryCtx, systemPrompt, history, userMessage, image, model)
		}); handled {
			return retryResult, retryErr
		}
		return "", err
	}
	respBodyClosed := false
	closeRespBody := func() {
		if respBodyClosed {
			return
		}
		respBodyClosed = true
		_ = resp.Body.Close()
	}
	defer closeRespBody()

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
	result, err := p.handleSSEResponse(ctx, resp, spinner, thinkingMsg, model)
	if err != nil {
		if retryResult, handled, retryErr := retryGeminiTimeout(ctx, err, "multimodal FC mode", func(retryCtx context.Context) (string, error) {
			closeRespBody()
			return p.ChatWithImage(retryCtx, systemPrompt, history, userMessage, image, model)
		}); handled {
			return retryResult, retryErr
		}
	}
	return result, err
}
