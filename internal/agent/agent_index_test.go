package agent

import (
	"sync"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/config"
)

// TestTriggerIndexUpdate_Debounce は3回連続呼び出しでタイマーが正しくリセットされることを検証
func TestTriggerIndexUpdate_Debounce(t *testing.T) {
	// テスト用にEmbeddingを有効化（元の値を保存して復元）
	cfg := config.GetGlobalConfig()
	originalEnabled := cfg.Embedding.Enabled
	originalBaseURL := cfg.Embedding.BaseURL
	defer func() {
		cfg.Embedding.Enabled = originalEnabled
		cfg.Embedding.BaseURL = originalBaseURL
	}()

	// テスト実行用に設定を上書き
	cfg.Embedding.Enabled = true
	cfg.Embedding.BaseURL = "http://invalid-url:9999" // Ollama接続エラーを回避

	// モックAgentを作成
	a := &Agent{
		indexDebounceMu: sync.Mutex{},
	}

	// トリガーを3回連続呼び出し
	for i := 0; i < 3; i++ {
		a.triggerIndexUpdate()
		time.Sleep(100 * time.Millisecond) // 少し間隔を空ける
	}

	// タイマーが設定されていることを確認
	a.indexDebounceMu.Lock()
	timer := a.indexDebounce
	a.indexDebounceMu.Unlock()

	if timer == nil {
		t.Error("debounce timer should be set")
	}

	// タイマーを停止（デバウンスが正しく動作していれば、まだ発火していないはず）
	a.indexDebounceMu.Lock()
	if a.indexDebounce != nil {
		a.indexDebounce.Stop()
		a.indexDebounce = nil
	}
	a.indexDebounceMu.Unlock()
}

// TestTriggerIndexUpdate_Disabled はEmbedding無効時にトリガーが何もしないことを確認
func TestTriggerIndexUpdate_Disabled(t *testing.T) {
	// テスト用にEmbeddingを無効化（元の値を保存して復元）
	cfg := config.GetGlobalConfig()
	originalEnabled := cfg.Embedding.Enabled
	defer func() {
		cfg.Embedding.Enabled = originalEnabled
	}()

	// テスト実行用に設定を上書き
	cfg.Embedding.Enabled = false

	// モックAgentを作成
	a := &Agent{
		indexDebounceMu: sync.Mutex{},
	}

	// トリガー呼び出し（Embedding無効時は何もしない）
	a.triggerIndexUpdate()

	// タイマーが設定されていないことを確認
	a.indexDebounceMu.Lock()
	timer := a.indexDebounce
	a.indexDebounceMu.Unlock()

	if timer != nil {
		t.Error("debounce timer should not be set when embedding is disabled")
	}
}
