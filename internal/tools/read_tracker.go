package tools

import "sync"

// ReadTracker はセッション内で read_file されたファイルパスを追跡する。
// 書き込み系ツール実行前に「このファイルを読んだか？」をチェックするために使用。
type ReadTracker struct {
	mu    sync.RWMutex
	paths map[string]bool // key: absPath
}

// GlobalReadTracker はグローバルな ReadTracker インスタンス。
// agent.NewAgent() と /clear コマンドで Reset される。
var GlobalReadTracker = NewReadTracker()

// NewReadTracker は新しい ReadTracker を作成する。
func NewReadTracker() *ReadTracker {
	return &ReadTracker{
		paths: make(map[string]bool),
	}
}

// MarkRead はファイルを「読み済み」として記録する（absPath で保存）。
func (rt *ReadTracker) MarkRead(absPath string) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.paths[absPath] = true
}

// IsRead はファイルが読み済みかを返す。
func (rt *ReadTracker) IsRead(absPath string) bool {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	return rt.paths[absPath]
}

// Reset はトラッカーをクリアする（新セッション開始時・/clear 時）。
func (rt *ReadTracker) Reset() {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.paths = make(map[string]bool)
}
