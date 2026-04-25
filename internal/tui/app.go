package tui

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/susugadx/xelyon-cli/internal/tui/lifecycle"
)

// Run は TUI モードでアプリケーションを起動する。
func Run(agent AgentInterface, initialContent string, onProgram func(*tea.Program)) {
	// defer は LIFO: runExitCallbacks (チャネル停止) → RestoreTerminal (画面復旧) の順で実行
	defer lifecycle.RestoreTerminal()
	defer lifecycle.RunExitCallbacks()

	m := NewModel(agent, initialContent)

	// マウスホイールを直接処理（mode 1007 経由のキー変換より低遅延）。
	// ネイティブのテキスト選択は Shift+ドラッグで可能。
	// コピペは /copy コマンドで対応。
	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if onProgram != nil {
		onProgram(p)
	}

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}
}
