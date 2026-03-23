package agent

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tui"
)

// RunTUIWithConfig は TUI モードでインタラクティブセッションを起動する。
func RunTUIWithConfig(model string, provider api.Provider, cfg *config.Config, autoApprove bool) {
	ag := InitInteractiveAgent(model, provider, cfg, autoApprove)
	defer ag.Cleanup()

	// TUIAdapter を作成（sendMsg は後で p.Send 経由で接続）
	adapter := NewTUIAdapter(ag, nil)

	tui.Run(adapter, func(p *tea.Program) {
		// capture writer に p.Send を接続
		adapter.sendMsg = func(msg tui.AppendMessageMsg) {
			p.Send(msg)
		}
		adapter.SetOutputCapture()
	})
}
