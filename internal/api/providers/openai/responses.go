package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

const defaultOpenAIResponsesURL = "https://api.openai.com/v1/responses"

// isCodexModel は Codex モデルかどうかを判定
// Codex モデルは reasoning が必須（"none" 非サポート）
func isCodexModel(model string) bool {
	return strings.Contains(strings.ToLower(model), "codex")
}

// convertHistoryToResponsesInput は api.ConvertHistoryToInputItems のラッパー
// Responses API 用の InputItem 形式に変換
func convertHistoryToResponsesInput(history []api.Message) []InputItem {
	return api.ConvertHistoryToInputItems(history)
}

// ReasoningConfig は OpenAI Extended Thinking の設定
type ReasoningConfig struct {
	Effort string `json:"effort"` // none, low, medium, high（omitempty削除で明示的に送信）
}

// ResponsesTool は Responses API 用のツール定義
// Chat Completions API と異なり、"function" キーのネストが不要
type ResponsesTool struct {
	Type        string                 `json:"type"`                  // "function"
	Name        string                 `json:"name"`                  // ツール名
	Description string                 `json:"description,omitempty"` // ツールの説明
	Parameters  map[string]interface{} `json:"parameters,omitempty"`  // JSON Schema
	Strict      bool                   `json:"strict,omitempty"`      // Structured Output
}

// ResponsesRequest は Responses API リクエスト
type ResponsesRequest struct {
	Model                string           `json:"model"`
	Input                interface{}      `json:"input,omitempty"`                // string or []InputItem（previous_response_id使用時は省略可）
	PreviousResponseID   string           `json:"previous_response_id,omitempty"` // 前回のレスポンスID（キャッシュ用）
	Instructions         string           `json:"instructions,omitempty"`         // システムプロンプト
	MaxOutputTokens      int              `json:"max_output_tokens,omitempty"`    // 最大出力トークン数
	Stream               bool             `json:"stream,omitempty"`
	Reasoning            *ReasoningConfig `json:"reasoning,omitempty"`              // Extended Thinking
	Tools                []ResponsesTool  `json:"tools,omitempty"`                  // ツール定義
	PromptCacheKey       string           `json:"prompt_cache_key,omitempty"`       // プロンプトキャッシュのルーティングキー
	PromptCacheRetention string           `json:"prompt_cache_retention,omitempty"` // キャッシュ保持期間（"24h"でextended cache）
}

// InputItem は api.InputItem のエイリアス（api packageで定義）
type InputItem = api.InputItem

// InputContentPart は api.InputContentPart のエイリアス（api packageで定義）
type InputContentPart = api.InputContentPart

// ResponseMetadata はレスポンスメタデータ（response.created / response.completed イベント用）
type ResponseMetadata struct {
	ID     string          `json:"id"`               // "resp_xxx..."
	Status string          `json:"status,omitempty"` // "in_progress", "completed"
	Model  string          `json:"model,omitempty"`
	Usage  *ResponsesUsage `json:"usage,omitempty"` // response.completed で取得
}

// ResponsesUsage は Responses API の usage 情報
type ResponsesUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// ResponsesError は Responses API のエラー情報
type ResponsesError struct {
	Type    string `json:"type"`              // "insufficient_quota", "rate_limit_exceeded" 等
	Code    string `json:"code"`              // エラーコード
	Message string `json:"message,omitempty"` // エラーメッセージ
}

// ResponsesStreamChunk は Responses API ストリーミングチャンク
type ResponsesStreamChunk struct {
	Type     string            `json:"type"`               // "response.output_text.delta", "response.created", etc.
	Delta    string            `json:"delta,omitempty"`    // テキスト差分
	Response *ResponseMetadata `json:"response,omitempty"` // response.created で取得
	Item     *ResponsesItem    `json:"item,omitempty"`     // response.output_item.added で取得（function_call用）
	Usage    *ResponsesUsage   `json:"usage,omitempty"`    // response.completed で取得
	Error    *ResponsesError   `json:"error,omitempty"`    // error イベントで取得
}

// ResponsesItem は output_item のデータ（function_call 等）
type ResponsesItem struct {
	Type      string `json:"type,omitempty"`      // "function_call"
	Name      string `json:"name,omitempty"`      // ツール名
	CallID    string `json:"call_id,omitempty"`   // 呼び出しID
	Arguments string `json:"arguments,omitempty"` // 完了時の引数（response.function_call_arguments.done）
}

// ResponsesResult は Responses API のレスポンス結果（ID付き）
type ResponsesResult struct {
	Content    string // テキストコンテンツ
	ResponseID string // レスポンスID（response.created から取得）
}

// chatWithResponses は Responses API でチャット
// previous_response_id を使用してキャッシュを活用
func (p *Provider) chatWithResponses(ctx context.Context, systemPrompt string, history []api.Message, model string) (string, error) {
	if os.Getenv("XELYON_DEBUG_OPENAI") == "1" {
		fmt.Fprintf(os.Stderr, "[DEBUG OpenAI] chatWithResponses called, model=%s\n", model)
	}
	cfg := config.GetGlobalConfig()

	// Responses API URL
	apiURL := os.Getenv("OPENAI_RESPONSES_URL")
	if apiURL == "" {
		apiURL = defaultOpenAIResponsesURL
	}

	reqBody := ResponsesRequest{
		Model:                model,
		MaxOutputTokens:      api.GetMaxOutputTokens("openai", 16384),
		Stream:               true,
		Tools:                GetResponsesToolDefinitions(p.mcpTools), // Function Calling
		PromptCacheKey:       "xelyon",
		PromptCacheRetention: "24h",
	}

	// システムプロンプトを developer メッセージとして Input の先頭に追加
	// （Instructions フィールドだと Prompt Cache が効かないため）
	developerMsg := InputItem{
		Type:    "message",
		Role:    "developer",
		Content: systemPrompt,
	}

	// previous_response_id がある場合はキャッシュを活用
	// ただし、Function Calling 結果の場合は full history を送信（function_call との対応が必要）
	if p.lastResponseID != "" && len(history) > 0 {
		lastMsg := history[len(history)-1]

		if lastMsg.Role == "tool" {
			// Function Calling 結果: previous_response_id を使わず full history
			// （function_call + function_call_output の対応が必要）
			historyInput := convertHistoryToResponsesInput(history)
			reqBody.Input = append([]InputItem{developerMsg}, historyInput...)
		} else {
			// 通常メッセージ: previous_response_id で最新メッセージのみ
			// NOTE: previous_response_id 使用時は developer メッセージ不要（前回と同じなので）
			reqBody.PreviousResponseID = p.lastResponseID
			reqBody.Input = []InputItem{{
				Type:    "message",
				Role:    lastMsg.Role,
				Content: lastMsg.Content,
			}}
		}
	} else {
		// 初回または responseID がない場合は履歴全体を送信
		historyInput := convertHistoryToResponsesInput(history)
		reqBody.Input = append([]InputItem{developerMsg}, historyInput...)
	}

	// Extended Thinking 適用
	if cfg.Thinking.Enabled {
		reqBody.Reasoning = &ReasoningConfig{
			Effort: LevelToReasoningEffort(cfg.Thinking.Level),
		}
	} else if isCodexModel(model) {
		// Codexモデルは reasoning 必須のため "low" にフォールバック
		reqBody.Reasoning = &ReasoningConfig{
			Effort: "low",
		}
	}
	// 非Codexモデルでthinking無効の場合はreasoningフィールドを送信しない（nil）

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// デバッグ: リクエストボディを出力
	if os.Getenv("XELYON_DEBUG_OPENAI") == "1" {
		fmt.Fprintf(os.Stderr, "[DEBUG OpenAI Responses] Request body:\n%s\n", string(jsonBody))
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	// スピナー開始
	spinner := api.StartThinkingSpinner(isCodexModel(model) && cfg.Thinking.Enabled, "")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		spinner.Stop()
		return "", fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	// previous_response_id が無効な場合のフォールバック
	if resp.StatusCode == http.StatusBadRequest && p.lastResponseID != "" {
		spinner.Stop()
		// responseID をクリアしてリトライ
		p.lastResponseID = ""
		return p.chatWithResponses(ctx, systemPrompt, history, model)
	}

	if resp.StatusCode != http.StatusOK {
		return "", api.HandleHTTPError(resp, spinner, p.Name())
	}

	content, responseID, err := p.handleResponsesStreaming(ctx, resp, spinner)
	if err == nil && responseID != "" {
		p.lastResponseID = responseID
	}
	return content, err
}

// responsesFunctionCallAccumulator は Responses API の function_call を累積
type responsesFunctionCallAccumulator struct {
	CallID    string
	Name      string
	Arguments strings.Builder
}

// handleResponsesStreaming は Responses API のストリーミングを処理
// Response ID も抽出して返却（content, responseID, error）
func (p *Provider) handleResponsesStreaming(ctx context.Context, resp *http.Response, spinner *ui.Spinner) (string, string, error) {
	var responseID string // response.created イベントから抽出

	// Function Calling: 累積用
	functionCalls := make(map[string]*responsesFunctionCallAccumulator) // call_id -> accumulator
	var toolCallsOutput strings.Builder

	// usage 情報を追跡
	var lastUsage *api.Usage

	parser := func(line string) (string, bool, error) {
		// デバッグ: 全SSE行を出力
		if os.Getenv("XELYON_DEBUG_OPENAI") == "1" && line != "" {
			fmt.Fprintf(os.Stderr, "[DEBUG OpenAI Responses] SSE line: %s\n", line)
		}

		// SSE形式: "event: xxx" と "data: {...}" の組み合わせ
		if !strings.HasPrefix(line, "data: ") {
			return "", false, nil
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			return "", true, nil
		}

		var chunk ResponsesStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			return "", false, nil // パースエラーはスキップ
		}

		// デバッグログ: イベントタイプを表示
		if os.Getenv("XELYON_DEBUG_OPENAI") == "1" {
			fmt.Fprintf(os.Stderr, "[DEBUG OpenAI Responses] event: %s\n", chunk.Type)
			// response.completed の生データを表示
			if chunk.Type == "response.completed" {
				fmt.Fprintf(os.Stderr, "[DEBUG OpenAI Responses] raw data: %s\n", data)
			}
		}

		// response.created イベントから Response ID を抽出
		if chunk.Type == "response.created" && chunk.Response != nil {
			responseID = chunk.Response.ID
		}

		// error イベント: API エラー（quota超過、レート制限など）
		if chunk.Type == "error" {
			errMsg := "OpenAI API error"
			if chunk.Error != nil {
				if chunk.Error.Message != "" {
					errMsg = chunk.Error.Message
				} else if chunk.Error.Code != "" {
					errMsg = fmt.Sprintf("OpenAI API error: %s", chunk.Error.Code)
				}
			}
			return "", true, fmt.Errorf("%s", errMsg)
		}

		// response.failed イベント: リクエスト失敗（error イベントのフォールバック）
		if chunk.Type == "response.failed" {
			return "", true, fmt.Errorf("OpenAI Responses API request failed")
		}

		// response.output_item.added: function_call 開始
		if chunk.Type == "response.output_item.added" && chunk.Item != nil && chunk.Item.Type == "function_call" {
			acc := &responsesFunctionCallAccumulator{
				CallID: chunk.Item.CallID,
				Name:   chunk.Item.Name,
			}
			functionCalls[chunk.Item.CallID] = acc
		}

		// response.function_call_arguments.delta: 引数の差分
		if chunk.Type == "response.function_call_arguments.delta" {
			callID := ""
			if chunk.Item != nil {
				callID = chunk.Item.CallID
			}

			if callID != "" {
				// call_id がある場合は対応するアキュムレータに追加
				if acc, ok := functionCalls[callID]; ok {
					acc.Arguments.WriteString(chunk.Delta)
				}
			} else if len(functionCalls) == 1 {
				// call_id が空で単一 function_call の場合のみ追加
				for _, acc := range functionCalls {
					acc.Arguments.WriteString(chunk.Delta)
					break
				}
			}
			// call_id が空で複数 function_call の場合は無視（どれに追加すべきか不明）
		}

		// response.function_call_arguments.done: 引数完了
		if chunk.Type == "response.function_call_arguments.done" && chunk.Item != nil {
			if acc, ok := functionCalls[chunk.Item.CallID]; ok {
				// 完了した引数で上書き（doneイベントには完全な引数が含まれる）
				if chunk.Item.Arguments != "" {
					acc.Arguments.Reset()
					acc.Arguments.WriteString(chunk.Item.Arguments)
				}
				// Arguments が空の場合は delta で累積した値をそのまま使用
			}
		}

		// response.output_text.delta でテキスト差分を取得
		if chunk.Type == "response.output_text.delta" {
			return chunk.Delta, false, nil
		}

		// response.completed または response.done で終了
		if chunk.Type == "response.completed" || chunk.Type == "response.done" {
			// usage 情報を抽出（response.completed では response.usage にネストされている）
			var usage *ResponsesUsage
			if chunk.Response != nil && chunk.Response.Usage != nil {
				usage = chunk.Response.Usage
			} else if chunk.Usage != nil {
				// フォールバック: トップレベルの usage もチェック
				usage = chunk.Usage
			}

			if usage != nil {
				lastUsage = &api.Usage{
					InputTokens:  usage.InputTokens,
					OutputTokens: usage.OutputTokens,
				}
				if os.Getenv("XELYON_DEBUG_OPENAI") == "1" {
					fmt.Fprintf(os.Stderr, "[DEBUG OpenAI Responses] usage received: input=%d, output=%d\n",
						usage.InputTokens, usage.OutputTokens)
				}
			} else if os.Getenv("XELYON_DEBUG_OPENAI") == "1" {
				fmt.Fprintf(os.Stderr, "[DEBUG OpenAI Responses] %s event but usage is nil\n", chunk.Type)
			}

			// Function Calling: 累積した呼び出しを内部形式に変換
			for _, acc := range functionCalls {
				tc := &api.OpenAIToolCall{
					ID:   acc.CallID,
					Type: "function",
					Function: api.OpenAIToolCallFunction{
						Name:      acc.Name,
						Arguments: acc.Arguments.String(),
					},
				}
				if toolJSON, err := ConvertToolCallToToolJSON(tc); err == nil {
					toolCallsOutput.WriteString(toolJSON)
				}
			}
			return "", true, nil
		}

		return "", false, nil
	}

	content, err := api.ParseStreamingResponse(ctx, resp, spinner, parser)
	if err != nil {
		return "", responseID, err
	}

	// usage コールバックを呼び出し
	if lastUsage != nil && p.usageCallback != nil {
		p.usageCallback(*lastUsage)
	}

	// tool_calls がある場合はそれを返す
	if toolCallsOutput.Len() > 0 {
		if content != "" {
			return content + toolCallsOutput.String(), responseID, nil
		}
		return toolCallsOutput.String(), responseID, nil
	}
	return content, responseID, nil
}

// chatWithImageResponses は Responses API で画像付きメッセージを処理
// NOTE: 画像付きの場合は previous_response_id を使用しない（キャッシュ動作が不明瞭なため）
func (p *Provider) chatWithImageResponses(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
	cfg := config.GetGlobalConfig()

	// Responses API URL
	apiURL := os.Getenv("OPENAI_RESPONSES_URL")
	if apiURL == "" {
		apiURL = defaultOpenAIResponsesURL
	}

	// システムプロンプトを developer メッセージとして Input の先頭に追加
	// （Instructions フィールドだと Prompt Cache が効かないため）
	developerMsg := InputItem{
		Type:    "message",
		Role:    "developer",
		Content: systemPrompt,
	}

	// 入力を構築（developer + 履歴）
	input := append([]InputItem{developerMsg}, convertHistoryToResponsesInput(history)...)

	// 画像付きユーザーメッセージを追加
	dataURL := fmt.Sprintf("data:%s;base64,%s", image.MediaType, image.Base64)
	imageMessage := InputItem{
		Type: "message",
		Role: "user",
		Content: []InputContentPart{
			{
				Type:     "input_image",
				ImageURL: dataURL,
			},
			{
				Type: "input_text",
				Text: userMessage,
			},
		},
	}
	input = append(input, imageMessage)

	reqBody := ResponsesRequest{
		Model:                model,
		Input:                input,
		MaxOutputTokens:      api.GetMaxOutputTokens("openai", 16384),
		Stream:               true,
		Tools:                GetResponsesToolDefinitions(p.mcpTools), // Function Calling
		PromptCacheKey:       "xelyon",
		PromptCacheRetention: "24h",
	}

	// Extended Thinking 適用
	if cfg.Thinking.Enabled {
		reqBody.Reasoning = &ReasoningConfig{
			Effort: LevelToReasoningEffort(cfg.Thinking.Level),
		}
	} else if isCodexModel(model) {
		// Codexモデルは reasoning 必須のため "low" にフォールバック
		reqBody.Reasoning = &ReasoningConfig{
			Effort: "low",
		}
	}
	// 非Codexモデルでthinking無効の場合はreasoningフィールドを送信しない（nil）

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// デバッグ: リクエストボディを出力
	if os.Getenv("XELYON_DEBUG_OPENAI") == "1" {
		fmt.Fprintf(os.Stderr, "[DEBUG OpenAI Responses] Request body:\n%s\n", string(jsonBody))
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	// スピナー開始
	spinner := api.StartThinkingSpinner(isCodexModel(model) && cfg.Thinking.Enabled, "")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		spinner.Stop()
		return "", fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", api.HandleHTTPError(resp, spinner, p.Name())
	}

	content, responseID, err := p.handleResponsesStreaming(ctx, resp, spinner)
	if err == nil && responseID != "" {
		// 画像メッセージ後も responseID を保存（次回テキストのみの場合に使用可能）
		p.lastResponseID = responseID
	}
	return content, err
}
