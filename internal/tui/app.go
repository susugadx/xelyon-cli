package tui

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

// Alternate Scroll Mode (DECSET 1007)
// Alt Screen 時にマウスホイールをカーソルキー(Up/Down)に変換するターミナル機能。
// マウストラッキング(mode 1000/1002/1003)を一切使わないため、
// クリック/ドラッグによるネイティブテキスト選択・コピペが完全に動作する。
// 対応: xterm, VTE (GNOME Terminal), Windows Terminal, iTerm2, kitty, Alacritty
const (
	enableAltScrollSeq  = "\x1b[?1007h"
	disableAltScrollSeq = "\x1b[?1007l"
)

// restoreTerminal は Alt Screen を抜けてターミナルを復旧する。
// panic や異常終了時に defer で呼ばれる。
func restoreTerminal() {
	fmt.Fprint(os.Stdout, disableAltScrollSeq+"\033[?1049l\033[?25h")
}

// Run は TUI モードでアプリケーションを起動する。
func Run(agent AgentInterface, initialContent string, onProgram func(*tea.Program)) {
	defer restoreTerminal()

	m := NewModel(agent, initialContent)

	// マウスオプションは一切使わない。
	// ホイールスクロールは Alternate Scroll Mode (1007) で
	// カーソルキーに変換され、viewport の KeyUp/KeyDown で処理される。
	// ネイティブのテキスト選択/コピペはそのまま動作する。
	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
	)

	if onProgram != nil {
		onProgram(p)
	}

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "TUI error: %v\n", err)
		os.Exit(1)
	}
}
