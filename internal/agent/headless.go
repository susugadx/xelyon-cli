package agent

import (
	"encoding/json"
	"time"
)

// HeadlessResult はHeadlessモードの実行結果
type HeadlessResult struct {
	Status     string           `json:"status"`               // "success" or "error"
	Provider   string           `json:"provider"`             // LLMプロバイダー名
	Model      string           `json:"model"`                // モデル名
	Response   string           `json:"response"`             // AIの最終回答
	ToolCalls  []ToolCallResult `json:"tool_calls,omitempty"` // 実行されたツール呼び出し
	Tokens     *TokenUsage      `json:"tokens,omitempty"`     // トークン使用量
	DurationMs int64            `json:"duration_ms"`          // 実行時間（ミリ秒）
	Timestamp  string           `json:"timestamp"`            // タイムスタンプ（RFC3339）
	Error      *ErrorInfo       `json:"error,omitempty"`      // エラー情報
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
	Input  int `json:"input"`  // 入力トークン数
	Output int `json:"output"` // 出力トークン数
	Total  int `json:"total"`  // 合計トークン数
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
		Status:     "success",
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
		Status:     "error",
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
