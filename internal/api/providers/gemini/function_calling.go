package gemini

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

// getGeminiFunctionCallingURL は Function Calling 用の URL を生成
// NOTE: Gemini APIはPretty-printed JSONを返すため、行単位ストリーミングは不可能
// 非ストリーミング（generateContent）エンドポイントを使用
func getGeminiFunctionCallingURL(model string) string {
	if baseURL := os.Getenv("GEMINI_API_URL"); baseURL != "" {
		return baseURL
	}
	return fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent", model)
}

// chatWithFunctionCalling は Function Calling API を使用してツールを呼び出す
func (p *Provider) chatWithFunctionCalling(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	debug := os.Getenv("XELYON_DEBUG_GEMINI") == "1"

	// モデル名を設定（config優先、フォールバックはgemini-2.0-flash）
	model = api.GetDefaultModel(model, "gemini", "gemini-2.0-flash")

	// メッセージを interface{} スライスに変換（Function Calling リクエスト用）
	var contents []interface{}

	// System prompt を最初のユーザーメッセージとして追加
	if systemPrompt != "" {
		contents = append(contents, GeminiContent{
			Parts: []GeminiPart{{Text: systemPrompt}},
			Role:  "user",
		})
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

	// Function Calling 用リクエストを構築（MCPツールを含める）
	reqBody := GeminiRequestWithTools{
		Contents: contents,
		Tools:    GetCombinedToolDefinitions(p.mcpTools),
	}

	// Thinking 設定（Gemini 3 vs 2.5 で自動分岐）
	reqBody.GenerationConfig = getThinkingConfigForModel(model, cfg)

	if debug && reqBody.GenerationConfig != nil && reqBody.GenerationConfig.ThinkingConfig != nil {
		tc := reqBody.GenerationConfig.ThinkingConfig
		if tc.ThinkingLevel != "" {
			fmt.Fprintf(os.Stderr, "[DEBUG Gemini FC] thinkingLevel=%q (Gemini 3)\n", tc.ThinkingLevel)
		} else {
			fmt.Fprintf(os.Stderr, "[DEBUG Gemini FC] thinkingBudget=%d (Gemini 2.5)\n", tc.ThinkingBudget)
		}
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	// Function Calling エンドポイント（非ストリーミング）
	url := getGeminiFunctionCallingURL(model)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", p.apiKey)

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

	resp, err := p.httpClient.Do(req)
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
			fmt.Fprintf(os.Stderr, "[DEBUG Gemini FC] Error response (status %d): %s\n", resp.StatusCode, bodyStr)
		}
		if rateLimitErr := api.HandleRateLimit(resp); rateLimitErr != nil {
			return "", rateLimitErr
		}
		if len(body) == 0 {
			return "", fmt.Errorf("gemini API error (status %d): empty response body", resp.StatusCode)
		}
		return "", api.SanitizeErrorMessage(body, resp.StatusCode)
	}

	// Function Calling レスポンスを処理（非ストリーミング JSON）
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		if spinner != nil {
			spinner.Stop()
		}
		return "", fmt.Errorf("failed to read FC response body: %w", err)
	}
	return p.handleFunctionCallingResponse(body, spinner)
}
