package tui

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

// restoreTerminal は Alt Screen を抜けてターミナルを復旧する。
// panic や異常終了時に defer で呼ばれる。
func restoreTerminal() {
	// Alt Screen 解除 + カーソル表示
	fmt.Fprint(os.Stdout, "\033[?1049l\033[?25h")
}

// Run は TUI モードでアプリケーションを起動する。
// agent は tui.AgentInterface を満たす必要がある。
// initialContent は起動時に viewport に表示する初期テキスト（ヘッダー等）。
// onProgram はProgram作成後に呼ばれるコールバック（出力キャプチャ設定用）。
func Run(agent AgentInterface, initialContent string, onProgram func(*tea.Program)) {
	// 異常終了時のターミナル復旧
	defer restoreTerminal()

	m := NewModel(agent, initialContent)

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
