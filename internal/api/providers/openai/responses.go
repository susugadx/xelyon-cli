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
	Store                bool             `json:"store"`                            // レスポンスを保存（previous_response_id に必要）
	Reasoning            *ReasoningConfig `json:"reasoning,omitempty"`              // Extended Thinking
	Tools                []ResponsesTool  `json:"tools,omitempty"`                  // ツール定義
	ToolChoice           interface{}      `json:"tool_choice,omitempty"`            // "auto", "none", "required", またはオブジェクト
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
	InputTokens         int                     `json:"input_tokens"`
	OutputTokens        int                     `json:"output_tokens"`
	InputTokensDetails  *ResponsesInputDetails  `json:"input_tokens_details,omitempty"`
	OutputTokensDetails *ResponsesOutputDetails `json:"output_tokens_details,omitempty"`
}

// ResponsesInputDetails は Responses API の入力トークン詳細
type ResponsesInputDetails struct {
	CachedTokens int `json:"cached_tokens,omitempty"`
}

// ResponsesOutputDetails は Responses API の出力トークン詳細
type ResponsesOutputDetails struct {
	ReasoningTokens int `json:"reasoning_tokens,omitempty"`
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
	errOut := api.ErrorWriterFromContext(ctx)
	if os.Getenv("XELYON_DEBUG_OPENAI") == "1" {
		fmt.Fprintf(errOut, "[DEBUG OpenAI] chatWithResponses called, model=%s\n", model)
	}

	apiURL := resolveResponsesAPIURL()
	reqBody := p.buildChatResponsesRequest(ctx, systemPrompt, history, model)

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// デバッグ: リクエストボディを出力
	if os.Getenv("XELYON_DEBUG_OPENAI") == "1" {
		fmt.Fprintf(errOut, "[DEBUG OpenAI Responses] Request body:\n%s\n", string(jsonBody))
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.APIKey)

	// スピナー開始
	spinner := api.StartThinkingSpinner(ctx, false, "", reqBody.Reasoning != nil)

	resp, err := p.ExecuteRequest(req)
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

// chatWithImageResponses は Responses API で画像付きメッセージを処理
// NOTE: 画像付きの場合は previous_response_id を使用しない（キャッシュ動作が不明瞭なため）
func (p *Provider) chatWithImageResponses(ctx context.Context, systemPrompt string, history []api.Message, userMessage string, image *api.ImageData, model string) (string, error) {
	apiURL := resolveResponsesAPIURL()
	reqBody := p.buildImageResponsesRequest(ctx, systemPrompt, history, userMessage, image, model)

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	// デバッグ: リクエストボディを出力
	if os.Getenv("XELYON_DEBUG_OPENAI") == "1" {
		errOut := api.ErrorWriterFromContext(ctx)
		fmt.Fprintf(errOut, "[DEBUG OpenAI Responses] Request body:\n%s\n", string(jsonBody))
	}

	req, err := http.NewRequestWithContext(ctx, "POST", apiURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.APIKey)

	// スピナー開始
	spinner := api.StartThinkingSpinner(ctx, false, "", reqBody.Reasoning != nil)

	resp, err := p.ExecuteRequest(req)
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
