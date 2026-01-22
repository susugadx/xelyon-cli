package config

import "time"

// HTTP Configuration
const (
	// DefaultHTTPTimeout は LLM API 用の HTTP クライアントタイムアウト
	// ストリーミング時はアイドルタイムアウト（streaming.idle_timeout_seconds）で管理するため、
	// HTTPクライアントレベルでは無制限（0）に設定
	DefaultHTTPTimeout = 0                // 無制限（context/アイドルタイムアウトで管理）
	SerperHTTPTimeout  = 10 * time.Second // Serper API専用（非ストリーミング）
)

// Tool Execution Limits
const (
	MaxToolIterations    = 25 // ツールループ最大回数（複雑なタスクに対応）
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

// Plan Execution Limits
const (
	PlanMaxRetries       = 3  // ステップ失敗時の最大リトライ回数
	PlanMaxIterations    = 10 // ステップ内ツール実行の最大イテレーション数
	PlanMaxAutoContinues = 3  // AI質問時の自動続行最大回数
)

// Message Processing Limits
const (
	MessageTruncateLen = 500 // サマリー生成時のメッセージ切り詰め長
	MaxLastOutputs     = 10  // 最後の出力履歴最大保存数
)

// UI Display Limits
const (
	HistoryPreviewLen     = 50  // /history コマンドのプレビュー切り詰め長
	SessionPreviewLen     = 60  // /sessions コマンドのプレビュー切り詰め長
	SessionListMaxDisplay = 10  // /sessions の最大表示数
	ErrorOutputMaxLines   = 20  // Plan失敗時のエラー出力最大行数
	SyncFileChangesMax    = 10  // /sync のファイル変更最大表示数
	DebugPreviewLen       = 500 // デバッグログのプレビュー長
)

// RepoMap Display Limits
const (
	RepoMapMaxFiles   = 20 // RepoMapの最大ファイル数
	RepoMapMaxSymbols = 10 // RepoMapのファイル毎最大シンボル数
)

// Test/Verification Limits
const (
	TestOutputMaxLines = 20 // テスト出力の最大行数
)

// Parallel Execution Defaults
const (
	DefaultParallelWorkers = 3 // Plan実行のデフォルト並列ワーカー数
	LoopDetectionThreshold = 3 // ループ検知の閾値
)
