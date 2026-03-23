package agent

import (
	"bytes"
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tui"
	"github.com/susugadx/xelyon-cli/internal/version"
)

// RunTUIWithConfig は TUI モードでインタラクティブセッションを起動する。
func RunTUIWithConfig(model string, provider api.Provider, cfg *config.Config, autoApprove bool) {
	ag := InitInteractiveAgent(model, provider, cfg, autoApprove)
	defer ag.Cleanup()

	// ヘッダー情報をキャプチャして初期コンテンツにする
	initialContent := buildTUIHeader(ag, model, provider, autoApprove)

	// TUIAdapter を作成（sendMsg は後で p.Send 経由で接続）
	adapter := NewTUIAdapter(ag, nil)

	tui.Run(adapter, initialContent, func(p *tea.Program) {
		// capture writer に p.Send を接続
		adapter.sendMsg = func(msg tui.AppendMessageMsg) {
			p.Send(msg)
		}
		adapter.SetOutputCapture()
	})
}

// buildTUIHeader は TUI 起動時に表示するヘッダー情報を構築する。
func buildTUIHeader(ag *Agent, model string, provider api.Provider, autoApprove bool) string {
	var buf bytes.Buffer

	// バージョン + モデル情報
	fmt.Fprintf(&buf, "🚀 XELYON CLI v%s\n", version.GetVersion())
	fmt.Fprintf(&buf, "   Provider: %s | Model: %s\n", provider.Name(), model)
	if autoApprove {
		fmt.Fprintf(&buf, "   Mode: Auto-approve (safe/medium)\n")
	}
	fmt.Fprintf(&buf, "\n")

	return buf.String()
}
