package tools

import "sync"

// ExecutionContext はツール実行時の周辺コンテキストを保持する。
// web_search などが現在のプロバイダー/モデルを参照するために使用する。
//
// 安全性:
//   - race-safe: sync.RWMutex で保護されており、複数 goroutine からの
//     同時読み書きでデータ競合は発生しない。
//   - ただし process-global な単一変数であるため、同一プロセス内で複数の Agent が
//     同時に異なる ExecutionContext を必要とする場合、意味的に正しい値が保証されない
//     （あるAgent の Set が別 Agent の Get に見える可能性がある）。
//   - 現状は single-agent CLI 前提で許容している。multi-agent 対応が必要になった場合は
//     Agent ごとに context を持つ設計への変更が必要。
//
// 並列実行（executeToolCallsWithParallel）での扱い:
//   - Phase 1a（parallel）: batch 開始前に Set、全 goroutine 完了後に Clear。
//     goroutine 内は GetExecutionContext()（RLock）で読み取りのみ。
//   - Phase 1b（sequential）: executeToolWithSpinner が個別に Set/Clear する。
type ExecutionContext struct {
	ProviderName string
	Model        string
}

var (
	executionContextMu sync.RWMutex
	executionContext   ExecutionContext
)

// SetExecutionContext は現在のツール実行コンテキストを更新する（race-safe）。
func SetExecutionContext(ctx ExecutionContext) {
	executionContextMu.Lock()
	defer executionContextMu.Unlock()
	executionContext = ctx
}

// ClearExecutionContext は現在のツール実行コンテキストをクリアする（race-safe）。
func ClearExecutionContext() {
	SetExecutionContext(ExecutionContext{})
}

// GetExecutionContext は現在のツール実行コンテキストを返す（race-safe、RLock）。
func GetExecutionContext() ExecutionContext {
	executionContextMu.RLock()
	defer executionContextMu.RUnlock()
	return executionContext
}
