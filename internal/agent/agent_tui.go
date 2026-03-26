package agent

import (
	"bytes"
	"sync/atomic"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools"
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

	// ツール結果チャネルを作成し、Agent に設定
	toolResultCh := make(chan tools.ToolResultInfo, 4096)
	ag.tuiToolResultCh = toolResultCh

	// TUIAdapter を作成（sendMsg は後で p.Send 経由で接続）
	adapter := NewTUIAdapter(ag, nil)

	tui.Run(adapter, initialContent, func(p *tea.Program) {
		// capture writer に p.Send を非同期チャネル経由で接続。
		// tea.Cmd goroutine 内から p.Send() を直接呼ぶとデッドロックするため、
		// バッファ付きチャネルと drain goroutine を経由させる。
		msgCh := make(chan tui.AppendMessageMsg, 4096)
		var closed atomic.Bool
		var dropCount atomic.Int64

		// drain goroutine: メッセージチャネルから読み出して p.Send() を呼ぶ
		go func() {
			for msg := range msgCh {
				p.Send(msg)
			}
		}()

		// drain goroutine: ツール結果チャネルから読み出して AppendToolResultMsg を p.Send()
		go func() {
			for info := range toolResultCh {
				if closed.Load() {
					return
				}
				summary := ui.FormatToolLine(ui.ToolDisplayInfo{
					ToolName: info.ToolName,
					Args:     info.Args,
					Result:   info.Result,
					Error:    info.Error,
				})
				p.Send(tui.AppendToolResultMsg{
					Tool: tui.ToolResult{
						Name:      info.ToolName,
						Summary:   summary,
						Detail:    info.Result,
						Collapsed: defaultToolCollapsed(info.ToolName, info.Result, info.Error),
						Error:     info.Error,
					},
				})
			}
		}()

		adapter.sendMsg = func(msg tui.AppendMessageMsg) {
			// closed フラグで closed channel への send panic を回避。
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
			ag.tuiToolResultClosed.Store(true)
			if n := dropCount.Load(); n > 0 {
				tui.DebugLog("TUI message channel: %d messages dropped", n)
			}
		})

		adapter.SetOutputCapture()
	})
}

// defaultToolCollapsed はツール種別に応じたデフォルトの折りたたみ状態を返す。
func defaultToolCollapsed(toolName, result string, isError bool) bool {
	// エラーは常に展開
	if isError {
		return false
	}

	switch toolName {
	case "apply_patch":
		return false // diff は見たい → 展開
	case "bash":
		return true // 成功は折りたたみ
	case "search_code", "read_file", "read_files":
		return true // 結果が長い → 折りたたみ
	case "web_search":
		return true // 結果が長い → 折りたたみ
	default:
		return true // デフォルトは折りたたみ
	}
}

// buildTUIHeader は TUI 起動時のグラデーションロゴヘッダーを返す。
func buildTUIHeader() string {
	return buildGradientHeader()
}
