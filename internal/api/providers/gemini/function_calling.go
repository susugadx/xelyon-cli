package gemini

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
)

// getGeminiFunctionCallingURL は非ストリーミング（generateContent）用の URL を生成
// ChatWithImage (FC有効時) で使用。chatWithFunctionCalling は SSE に移行済み。
func getGeminiFunctionCallingURL(model string) string {
	if baseURL := os.Getenv("GEMINI_API_URL"); baseURL != "" {
		return baseURL
	}
	return fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent", model)
}

// chatWithFunctionCalling は Function Calling API を使用してツールを呼び出す
func (p *Provider) chatWithFunctionCalling(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	debug := os.Getenv("XELYON_DEBUG_GEMINI") == "1"
	errOut := api.ErrorWriterFromContext(ctx)

	// モデル名を設定（config優先、フォールバックはgemini-3.1-pro-preview-customtools）
	// customtools版はカスタムツール優先度が高くパラレルFCを出す
	model = api.GetDefaultModelWithContext(ctx, model, "gemini", "gemini-3.1-pro-preview-customtools")

	// ツール定義を事前に取得（キャッシュにも含めるため）
	toolDefs := GetCombinedToolDefinitionsWithContext(ctx, p.mcpTools)
	fcMode := os.Getenv("GEMINI_FC_MODE")
	if fcMode == "" {
		fcMode = "AUTO"
	}
	toolCfg := &GeminiToolConfigWrapper{
		FunctionCallingConfig: GeminiFunctionCallingConfig{Mode: fcMode},
	}

	// キャッシュ管理（ツール定義もキャッシュに含める）
	cacheName, msgsToSend, err := p.updateOrUseCache(ctx, systemPrompt, history, model, toolDefs, toolCfg)
	if err != nil {
		return "", err
	}

	cfg := config.FromContext(ctx)

	// Function Calling 用リクエストを構築
	reqBody := GeminiRequestWithTools{
		Contents: geminiFunctionHistoryContents(msgsToSend, includeEmptyTextHistoryPart),
	}
	if cacheName != "" {
		// キャッシュ使用時: system_instruction, tools, tool_config はキャッシュに含まれているため除外
		reqBody.CachedContent = cacheName
	} else {
		if systemPrompt != "" {
			reqBody.SystemInstruction = &GeminiSystemInstruction{
				Parts: []GeminiPart{{Text: systemPrompt}},
			}
		}
		reqBody.Tools = toolDefs
		reqBody.ToolConfig = toolCfg
	}

	// Thinking 設定（Gemini 3 vs 2.5 で自動分岐）
	reqBody.GenerationConfig = getThinkingConfigForModel(ctx, model, cfg)

	if debug && reqBody.GenerationConfig != nil && reqBody.GenerationConfig.ThinkingConfig != nil {
		tc := reqBody.GenerationConfig.ThinkingConfig
		if tc.ThinkingLevel != "" {
			fmt.Fprintf(errOut, "[DEBUG Gemini FC] thinkingLevel=%q (Gemini 3)\n", tc.ThinkingLevel)
		} else {
			fmt.Fprintf(errOut, "[DEBUG Gemini FC] thinkingBudget=%d (Gemini 2.5)\n", tc.ThinkingBudget)
		}
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	// Context window overflow 可視化（debug のみ）
	if debug && len(jsonBody) > 500_000 {
		fmt.Fprintf(errOut, "[DEBUG Gemini FC] Large request: %d bytes (~%dk tokens)\n",
			len(jsonBody), len(jsonBody)/4/1000)
	}

	// Function Calling エンドポイント（SSEストリーミング）
	url := getGeminiURL(model)

	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", p.apiKey)

	// スピナー開始: レスポンス開始前は "Waiting for Gemini..." を表示
	// SSEストリーム開始後に thinking メッセージに切り替える
	thinkingMsg := getThinkingSpinnerMessage(ctx, model, false)
	spinner := api.StartSpinnerWithMessage(ctx, "Waiting for Gemini...")

	resp, err := p.doRequestWithRetry(ctx, req, jsonBody)
	if err != nil {
		if spinner != nil {
			spinner.Stop()
		}
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		if spinner != nil {
			spinner.Stop()
		}
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			return "", fmt.Errorf("gemini API error (status %d): unable to read response body - %v", resp.StatusCode, err)
		}
		if debug {
			bodyStr := string(body)
			if len(bodyStr) > 500 {
				bodyStr = bodyStr[:500] + "..."
			}
			fmt.Fprintf(errOut, "[DEBUG Gemini FC] Error response (status %d): %s\n", resp.StatusCode, bodyStr)
		}
		if rateLimitErr := api.HandleRateLimit(resp); rateLimitErr != nil {
			return "", rateLimitErr
		}
		// キャッシュ期限切れ検出 → 該当モデルのキャッシュを無効化してリトライ
		if cacheName != "" && isCacheExpiredError(resp.StatusCode, body) {
			p.invalidateCacheForRequest(ctx, model)
			if ctx.Value(cacheRetryKey) != nil {
				return "", fmt.Errorf("cache retry failed (status %d): %s", resp.StatusCode, string(body))
			}
			if debug {
				fmt.Fprintf(errOut, "[DEBUG Gemini FC] Cache expired, invalidating and retrying...\n")
			}
			ctx = context.WithValue(ctx, cacheRetryKey, true)
			return p.chatWithFunctionCalling(ctx, systemPrompt, history, model)
		}
		if len(body) == 0 {
			return "", fmt.Errorf("gemini API error (status %d): empty response body", resp.StatusCode)
		}
		return "", api.SanitizeErrorMessage(body, resp.StatusCode)
	}

	// Function Calling レスポンスを処理（SSE ストリーミング）
	return p.handleSSEResponse(ctx, resp, spinner, thinkingMsg)
}
