package tui

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

// Run は TUI モードでアプリケーションを起動する。
// agent は tui.AgentInterface を満たす必要がある。
// onProgram はProgram作成後に呼ばれるコールバック（出力キャプチャ設定用）。
func Run(agent AgentInterface, onProgram func(*tea.Program)) {
	m := NewModel(agent)

	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
		tea.WithMouseAllMotion(),
	)

	// Program 作成後のセットアップ（capture writer に p.Send を接続等）
	if onProgram != nil {
		onProgram(p)
	}

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}
}
