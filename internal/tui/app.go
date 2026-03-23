package tui

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
)

// マウスモード制御用エスケープシーケンス
const (
	// mode 1000: Normal Mouse Tracking — クリック/リリース/ホイールのみ（モーション追跡なし）
	// mode 1006: SGR Extended Mouse Mode — 座標上限を拡張
	enableMouseWheelSeq  = "\x1b[?1000h\x1b[?1006h"
	disableMouseWheelSeq = "\x1b[?1000l\x1b[?1006l"
)

// restoreTerminal は Alt Screen を抜けてターミナルを復旧する。
// panic や異常終了時に defer で呼ばれる。
func restoreTerminal() {
	// マウスモード解除 + Alt Screen 解除 + カーソル表示
	fmt.Fprint(os.Stdout, disableMouseWheelSeq+"\033[?1049l\033[?25h")
}

// Run は TUI モードでアプリケーションを起動する。
// agent は tui.AgentInterface を満たす必要がある。
// initialContent は起動時に viewport に表示する初期テキスト（ヘッダー等）。
// onProgram はProgram作成後に呼ばれるコールバック（出力キャプチャ設定用）。
func Run(agent AgentInterface, initialContent string, onProgram func(*tea.Program)) {
	// 異常終了時のターミナル復旧
	defer restoreTerminal()

	m := NewModel(agent, initialContent)

	// WithMouseAllMotion/WithMouseCellMotion は使わない。
	// これらは mode 1003/1002 を有効にし、全マウスイベントをキャプチャするため
	// ネイティブのテキスト選択/コピペが動かなくなる。
	// 代わりに mode 1000 (Normal Mouse Tracking) を手動で有効化し、
	// bubbletea のパーサーに SGR シーケンスを自動パースさせる。
	// ホイールイベントのみ Update で処理し、クリック/リリースは無視する。
	// テキスト選択は Shift+ドラッグで動作する（Windows Terminal/iTerm2 等）。
	p := tea.NewProgram(
		m,
		tea.WithAltScreen(),
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
