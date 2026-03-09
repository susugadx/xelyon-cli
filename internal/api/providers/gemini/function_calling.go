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
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/ui"
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

	// メッセージを interface{} スライスに変換（Function Calling リクエスト用）
	var contents []interface{}

	// 会話履歴を変換（ネイティブ functionCall / functionResponse 形式）
	// msgsToSend（差分のみ、または全量）を使用
	for _, msg := range msgsToSend {
		switch {
		case msg.Role == "assistant" && len(msg.ToolCalls) > 0:
			// assistant の functionCall パート
			parts := make([]interface{}, 0, len(msg.ToolCalls)+1)
			if msg.Content != "" {
				parts = append(parts, GeminiPart{Text: msg.Content})
			}
			// Gemini 3: thought パートを先に追加（最初の ToolCall から取得、全 TC で共有）
			if len(msg.ToolCalls) > 0 && len(msg.ToolCalls[0].ThoughtParts) > 0 {
				for _, tp := range msg.ToolCalls[0].ThoughtParts {
					geminiPart := make(map[string]any)
					if text, ok := tp["text"].(string); ok && text != "" {
						geminiPart["text"] = text
					}
					if thought, ok := tp["thought"].(bool); ok && thought {
						geminiPart["thought"] = true
					}
					if sig, ok := tp["thought_signature"].(string); ok && sig != "" {
						geminiPart["thoughtSignature"] = sig // Gemini API は camelCase
					}
					if len(geminiPart) > 0 {
						parts = append(parts, geminiPart)
					}
				}
			}
			for _, tc := range msg.ToolCalls {
				var args map[string]any
				_ = json.Unmarshal([]byte(tc.Function.Arguments), &args)
				parts = append(parts, GeminiFunctionCallPart{
					FunctionCall: GeminiFunctionCallData{
						Name: tc.Function.Name,
						Args: args,
					},
					ThoughtSignature: tc.ThoughtSignature,
				})
			}
			contents = append(contents, GeminiGenericContent{
				Parts: parts,
				Role:  "model",
			})

		case msg.Role == "tool" && msg.ToolCallID != "":
			// functionResponse パート（role: "user" — Gemini API仕様）
			toolName := msg.ToolName
			if toolName == "" {
				toolName = extractToolNameFromContent(msg.Content)
			}
			contents = append(contents, GeminiGenericContent{
				Parts: []interface{}{
					GeminiFunctionResponsePart{
						FunctionResponse: GeminiFunctionResponseData{
							Name: toolName,
							Response: map[string]any{
								"result": msg.Content,
							},
						},
					},
				},
				Role: "user",
			})

		default:
			// 通常のテキストメッセージ
			role := "user"
			if msg.Role == "assistant" {
				role = "model"
			}
			contents = append(contents, GeminiContent{
				Parts: []GeminiPart{{Text: msg.Content}},
				Role:  role,
			})
		}
	}

	cfg := tools.ConfigFromContext(ctx)

	// Function Calling 用リクエストを構築
	reqBody := GeminiRequestWithTools{
		Contents: contents,
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
		spinner = api.StartSpinnerWithMessage(ctx, msg)
	} else {
		spinner = api.StartThinkingSpinner(ctx, false, "")
	}

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
			p.invalidateCacheForModel(model)
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
	return p.handleSSEResponse(ctx, resp, spinner)
}

// extractToolNameFromContent はメッセージ内容からツール名を推定
func extractToolNameFromContent(content string) string {
	if strings.HasPrefix(content, "[Tool Result for ") {
		end := strings.Index(content, "]")
		if end > 17 {
			return content[17:end]
		}
	}
	return "unknown_tool"
}
