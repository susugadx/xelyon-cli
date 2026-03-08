package tools

import "sync"

// ExecutionContext はツール実行時の周辺コンテキストを保持する。
// web_search などが現在のプロバイダー/モデルを参照するために使用する。
//
// process-global 変数として保持されるが、sync.RWMutex で保護されているため goroutine-safe。
//
// 並列実行（executeToolCallsWithParallel）での扱い:
//   - Phase 1a（parallel）: batch 開始前に Set、全 goroutine 完了後に Clear。
//     goroutine 内は GetExecutionContext()（RLock）で読み取りのみ。
//   - Phase 1b（sequential）: executeToolWithSpinner が個別に Set/Clear する。
//   - 同一プロセス内で複数の Agent が同時に executeToolCallsWithParallel を
//     呼ぶケースは現状想定していない（CLI は single-agent）。
type ExecutionContext struct {
	ProviderName string
	Model        string
}

var (
	executionContextMu sync.RWMutex
	executionContext   ExecutionContext
)

// SetExecutionContext は現在のツール実行コンテキストを更新する（goroutine-safe）。
func SetExecutionContext(ctx ExecutionContext) {
	executionContextMu.Lock()
	defer executionContextMu.Unlock()
	executionContext = ctx
}

// ClearExecutionContext は現在のツール実行コンテキストをクリアする（goroutine-safe）。
func ClearExecutionContext() {
	SetExecutionContext(ExecutionContext{})
}

// GetExecutionContext は現在のツール実行コンテキストを返す（goroutine-safe、RLock）。
func GetExecutionContext() ExecutionContext {
	executionContextMu.RLock()
	defer executionContextMu.RUnlock()
	return executionContext
}
