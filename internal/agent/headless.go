package agent

import (
	"encoding/json"
	"fmt"
	"time"
)

const (
	// HeadlessStatusSuccess は headless JSON の成功 status。
	HeadlessStatusSuccess = "success"
	// HeadlessStatusError は headless JSON の失敗 status。
	HeadlessStatusError = "error"

	// HeadlessErrorTypeCancelled は context cancel / timeout 系の headless error type。
	HeadlessErrorTypeCancelled = "cancelled"
	// HeadlessErrorTypeAPI は provider request 失敗の headless error type。
	HeadlessErrorTypeAPI = "api_error"
	// HeadlessErrorTypeProviderSetupRequired は provider credential setup 未完了の headless error type。
	HeadlessErrorTypeProviderSetupRequired = "provider_setup_required"
	// HeadlessErrorTypeToolLoopLimit は tool loop limit 到達時の headless error type。
	HeadlessErrorTypeToolLoopLimit = "tool_loop_limit"
)

// HeadlessResult はHeadlessモードの実行結果
type HeadlessResult struct {
	Status             string           `json:"status"`                        // HeadlessStatusSuccess or HeadlessStatusError
	Provider           string           `json:"provider"`                      // LLMプロバイダー名
	Model              string           `json:"model"`                         // モデル名
	Response           string           `json:"response"`                      // AIの最終回答
	ToolCalls          []ToolCallResult `json:"tool_calls,omitempty"`          // 実行されたツール呼び出し
	Tokens             *TokenUsage      `json:"tokens,omitempty"`              // トークン使用量
	WebSearch          *WebSearchUsage  `json:"web_search,omitempty"`          // ネイティブ Web 検索の固定料金観測
	DurationMs         int64            `json:"duration_ms"`                   // 実行時間（ミリ秒）
	Timestamp          string           `json:"timestamp"`                     // タイムスタンプ（RFC3339）
	Error              *ErrorInfo       `json:"error,omitempty"`               // エラー情報
	Cost               float64          `json:"cost"`                          // 推定コスト（USD）
	PricingUnavailable bool             `json:"pricing_unavailable,omitempty"` // 既知の料金表がない場合 true
}

// ToolCallResult は個別のツール呼び出し結果
type ToolCallResult struct {
	Tool    string            `json:"tool"`    // ツール名
	Args    map[string]string `json:"args"`    // 引数
	Output  string            `json:"output"`  // 出力
	Success bool              `json:"success"` // 成功フラグ
}

// TokenUsage はトークン使用量
type TokenUsage struct {
	Input    int `json:"input"`    // 入力トークン数
	Cached   int `json:"cached"`   // キャッシュヒット入力トークン数
	Output   int `json:"output"`   // 出力トークン数
	Thinking int `json:"thinking"` // Thinking トークン数
	Total    int `json:"total"`    // 合計トークン数
}

// WebSearchUsage はネイティブ Web 検索の call fee と検索結果 token 観測を表す。
type WebSearchUsage struct {
	Calls        int     `json:"calls"`                   // built-in web search 呼び出し回数
	FeeEstimate  float64 `json:"fee_estimate"`            // 推定 call fee（USD）
	ResultTokens int     `json:"result_tokens,omitempty"` // provider が返した検索結果 token 観測値
}

// ErrorInfo はエラー情報
type ErrorInfo struct {
	Type    string `json:"type"`           // エラータイプ（api_error, tool_error, etc.）
	Message string `json:"message"`        // エラーメッセージ
	Code    int    `json:"code,omitempty"` // エラーコード（HTTPステータスなど）
}

// ToJSON は HeadlessResult を JSON 文字列に変換
func (r *HeadlessResult) ToJSON() (string, error) {
	bytes, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}

// NewSuccessResult は成功結果を生成
func NewSuccessResult(provider, model, response string, toolCalls []ToolCallResult, durationMs int64) *HeadlessResult {
	return &HeadlessResult{
		Status:     HeadlessStatusSuccess,
		Provider:   provider,
		Model:      model,
		Response:   response,
		ToolCalls:  toolCalls,
		DurationMs: durationMs,
		Timestamp:  time.Now().Format(time.RFC3339),
	}
}

// NewErrorResult はエラー結果を生成
func NewErrorResult(provider, model string, errType, errMsg string, durationMs int64) *HeadlessResult {
	return &HeadlessResult{
		Status:     HeadlessStatusError,
		Provider:   provider,
		Model:      model,
		Response:   "",
		DurationMs: durationMs,
		Timestamp:  time.Now().Format(time.RFC3339),
		Error: &ErrorInfo{
			Type:    errType,
			Message: errMsg,
		},
	}
}

// NewToolLoopLimitResult は headless tool loop limit 到達時の結果を生成する。
func NewToolLoopLimitResult(provider, model string, limit int, toolCalls []ToolCallResult, durationMs int64) *HeadlessResult {
	result := NewErrorResult(provider, model, HeadlessErrorTypeToolLoopLimit, HeadlessToolLoopLimitMessage(limit), durationMs)
	result.ToolCalls = toolCalls
	return result
}

// HeadlessToolLoopLimitMessage は tool loop limit 到達時のユーザー向け error message を返す。
func HeadlessToolLoopLimitMessage(limit int) string {
	return fmt.Sprintf("tool loop limit reached (%d iterations)", limit)
}
