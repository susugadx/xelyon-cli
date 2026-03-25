package agent

import (
	"bytes"
	"sync/atomic"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tui"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

// tuiAutoApproveReader は TUI モードで stdin を読む確認ダイアログに対して
// 常に "y\n" を返す io.Reader。bubbletea が stdin を占有するため、
// 全ツール確認を自動承認する。
type tuiAutoApproveReader struct{}

func (tuiAutoApproveReader) Read(p []byte) (int, error) {
	data := []byte("y\n")
	n := copy(p, data)
	return n, nil
}

// RunTUIWithConfig は TUI モードでインタラクティブセッションを起動する。
func RunTUIWithConfig(model string, provider api.Provider, cfg *config.Config, autoApprove bool) {
	// TUI モードでは確認ダイアログが動作しないため、全ツールを auto-approve する。
	// - autoApprove=true: apply_patch, write_file 等の ConfirmWithAutoApproveDecisionAndOptions 系
	// - tuiAutoApproveReader: bash の ConfirmWithIO（stdin から直接読む）系
	// Phase 2 で確認ダイアログを bubbletea 内に実装した後、bash の確認を復活させる。

	// Agent 初期化中の stdout 出力をキャプチャするバッファ。
	// Normal Screen に何も残さず、全てを Alt Screen の viewport に表示する。
	var captureBuf bytes.Buffer
	runtime := NewAgentRuntimeWithConfig(cfg)
	runtime.UI = ui.NewRuntime(tuiAutoApproveReader{}, &captureBuf, &captureBuf)
	runtime.AutoApprove = true

	ag := initInteractiveAgentWithRuntime(runtime, model, provider, true)
	defer ag.Cleanup()

	// SIGTERM 時に Alt Screen を復旧するフックを登録
	ag.exitHook = tui.RestoreTerminal

	// ヘッダー + キャプチャした初期化出力を結合して初期コンテンツにする
	initialContent := buildTUIHeader() + captureBuf.String()

	// TUIAdapter を作成（sendMsg は後で p.Send 経由で接続）
	adapter := NewTUIAdapter(ag, nil)

	tui.Run(adapter, initialContent, func(p *tea.Program) {
		// capture writer に p.Send を非同期チャネル経由で接続。
		// tea.Cmd goroutine 内から p.Send() を直接呼ぶとデッドロックするため、
		// バッファ付きチャネルと drain goroutine を経由させる。
		msgCh := make(chan tui.AppendMessageMsg, 4096)
		var closed atomic.Bool
		var dropCount atomic.Int64

		// drain goroutine: チャネルから読み出して p.Send() を呼ぶ
		go func() {
			for msg := range msgCh {
				p.Send(msg)
			}
		}()

		adapter.sendMsg = func(msg tui.AppendMessageMsg) {
			// closed フラグで closed channel への send panic を回避。
			// close(msgCh) ではなくフラグで制御し、drain goroutine は
			// プロセス終了で自然に回収される。
			if closed.Load() {
				return
			}
			select {
			case msgCh <- msg:
			default:
				dropCount.Add(1)
			}
		}

		// tui.Run 終了時: sendMsg を停止し、ドロップ統計をログ出力
		tui.OnExit(func() {
			closed.Store(true)
			if n := dropCount.Load(); n > 0 {
				tui.DebugLog("TUI message channel: %d messages dropped", n)
			}
		})

		adapter.SetOutputCapture()
	})
}

// buildTUIHeader は TUI 起動時のグラデーションロゴヘッダーを返す。
func buildTUIHeader() string {
	return buildGradientHeader()
}
