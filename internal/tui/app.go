package tui

import (
	"fmt"
	"os"
	"sync"

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

// exitCallbacks は Run 終了時に呼ばれるコールバック群
var (
	exitMu        sync.Mutex
	exitCallbacks []func()
)

// OnExit は TUI 終了時に呼ばれるコールバックを登録する。
// チャネルのクローズ等のリソース解放に使用。
func OnExit(fn func()) {
	exitMu.Lock()
	defer exitMu.Unlock()
	exitCallbacks = append(exitCallbacks, fn)
}

func runExitCallbacks() {
	exitMu.Lock()
	cbs := exitCallbacks
	exitCallbacks = nil
	exitMu.Unlock()
	for _, fn := range cbs {
		fn()
	}
}

// RestoreTerminal は Alt Screen を抜けてターミナルを復旧する。
// SIGTERM ハンドラ等の外部から呼べるように公開。
func RestoreTerminal() {
	fmt.Fprint(os.Stdout, disableAltScrollSeq+"\033[?1049l\033[?25h")
}

// DebugLog は TUI デバッグログに出力する。外部パッケージから呼べるように公開。
func DebugLog(format string, args ...any) {
	tuiDebugf(format, args...)
}

// Run は TUI モードでアプリケーションを起動する。
func Run(agent AgentInterface, initialContent string, onProgram func(*tea.Program)) {
	// defer は LIFO: runExitCallbacks (チャネル停止) → RestoreTerminal (画面復旧) の順で実行
	defer RestoreTerminal()
	defer runExitCallbacks()

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
