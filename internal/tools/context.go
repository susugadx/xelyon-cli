package tools

import "sync"

// ExecutionContext はツール実行時の周辺コンテキストを保持する。
// web_search などが現在のプロバイダー/モデルを参照するために使用する。
type ExecutionContext struct {
	ProviderName string
	Model        string
}

var (
	executionContextMu sync.RWMutex
	executionContext   ExecutionContext
)

// SetExecutionContext は現在のツール実行コンテキストを更新する。
func SetExecutionContext(ctx ExecutionContext) {
	executionContextMu.Lock()
	defer executionContextMu.Unlock()
	executionContext = ctx
}

// ClearExecutionContext は現在のツール実行コンテキストをクリアする。
func ClearExecutionContext() {
	SetExecutionContext(ExecutionContext{})
}

// GetExecutionContext は現在のツール実行コンテキストを返す。
func GetExecutionContext() ExecutionContext {
	executionContextMu.RLock()
	defer executionContextMu.RUnlock()
	return executionContext
}
