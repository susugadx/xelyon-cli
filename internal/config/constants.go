package config

import "time"

// HTTP Configuration
const (
	// DefaultHTTPTimeout は LLM API 用の HTTP クライアントタイムアウト
	// ストリーミング時はアイドルタイムアウト（streaming.idle_timeout_seconds）で管理するため、
	// HTTPクライアントレベルでは無制限（0）に設定
	DefaultHTTPTimeout = 0 // 無制限（context/アイドルタイムアウトで管理）
	SerperHTTPTimeout  = 10 * time.Second // Serper API専用（非ストリーミング）
)

// Tool Execution Limits
const (
	MaxToolIterations    = 10 // ツールループ最大回数
	MaxChangeStack       = 10 // Undo履歴最大保存数
	MaxAPIRetries        = 2  // API呼び出し最大リトライ回数
	MaxSameToolCallCount = 3  // 同じツール呼び出しの最大繰り返し回数（ループ検知）
)

// Output Display Limits
const (
	OutputTruncateLen   = 5000 // bash出力切り詰め長
	MaxDiffDisplayLines = 15   // diff表示最大行数
	MaxDiffIterations   = 20   // diff比較最大イテレーション
)
