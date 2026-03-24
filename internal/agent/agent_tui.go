package agent

import (
	"bytes"
	"fmt"
	"sync/atomic"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tui"
	"github.com/susugadx/xelyon-cli/internal/ui"
	"github.com/susugadx/xelyon-cli/internal/version"
)

// RunTUIWithConfig は TUI モードでインタラクティブセッションを起動する。
func RunTUIWithConfig(model string, provider api.Provider, cfg *config.Config, autoApprove bool) {
	// Agent 初期化中の stdout 出力をキャプチャするバッファ。
	// Normal Screen に何も残さず、全てを Alt Screen の viewport に表示する。
	var captureBuf bytes.Buffer
	runtime := NewAgentRuntimeWithConfig(cfg)
	runtime.UI = ui.NewRuntime(nil, &captureBuf, &captureBuf)

	ag := initInteractiveAgentWithRuntime(runtime, model, provider, autoApprove)
	defer ag.Cleanup()

	// SIGTERM 時に Alt Screen を復旧するフックを登録（P1-1対応）
	ag.exitHook = tui.RestoreTerminal

	// ヘッダー + キャプチャした初期化出力を結合して初期コンテンツにする
	initialContent := buildTUIHeader(model, provider, autoApprove) + captureBuf.String()

	// TUIAdapter を作成（sendMsg は後で p.Send 経由で接続）
	adapter := NewTUIAdapter(ag, nil)

	tui.Run(adapter, initialContent, func(p *tea.Program) {
		// capture writer に p.Send を非同期チャネル経由で接続。
		// tea.Cmd goroutine 内から p.Send() を直接呼ぶとデッドロックするため、
		// バッファ付きチャネルと drain goroutine を経由させる。
		msgCh := make(chan tui.AppendMessageMsg, 4096)
		var dropCount atomic.Int64

		// drain goroutine: チャネルから読み出して p.Send() を呼ぶ
		go func() {
			for msg := range msgCh {
				p.Send(msg)
			}
		}()

		// tui.Run 終了時に drain goroutine を停止（P1-2対応）
		tui.OnExit(func() {
			close(msgCh)
			if n := dropCount.Load(); n > 0 {
				tuiDebugf("TUI message channel: %d messages dropped", n)
			}
		})

		adapter.sendMsg = func(msg tui.AppendMessageMsg) {
			select {
			case msgCh <- msg:
			default:
				dropCount.Add(1)
			}
		}
		adapter.SetOutputCapture()
	})
}

// tuiDebugf は TUI デバッグログに出力する（agent パッケージから呼ぶ用）
func tuiDebugf(format string, args ...any) {
	tui.DebugLog(format, args...)
}

// buildTUIHeader は TUI 起動時に表示するヘッダー情報を構築する。
func buildTUIHeader(model string, provider api.Provider, autoApprove bool) string {
	var buf bytes.Buffer

	fmt.Fprintf(&buf, "🚀 XELYON CLI v%s\n", version.GetVersion())
	fmt.Fprintf(&buf, "   Provider: %s | Model: %s\n", provider.Name(), model)
	if autoApprove {
		fmt.Fprintf(&buf, "   Mode: Auto-approve (safe/medium)\n")
	}
	fmt.Fprintf(&buf, "\n")

	return buf.String()
}
