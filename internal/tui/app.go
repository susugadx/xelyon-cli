package tui

import (
	"fmt"
	"os"
	"sync"

	tea "github.com/charmbracelet/bubbletea"
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
	fmt.Fprint(os.Stdout, "\033[?1049l\033[?25h")
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
