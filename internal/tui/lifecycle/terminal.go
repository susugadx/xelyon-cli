package lifecycle

import (
	"fmt"
	"os"
	"sync"
)

var (
	exitMu        sync.Mutex
	exitCallbacks []func()
)

// OnExit は TUI 終了時に呼ばれるコールバックを登録する。
func OnExit(fn func()) {
	exitMu.Lock()
	defer exitMu.Unlock()
	exitCallbacks = append(exitCallbacks, fn)
}

// RunExitCallbacks は登録済みの終了コールバックを実行して登録をクリアする。
func RunExitCallbacks() {
	exitMu.Lock()
	cbs := exitCallbacks
	exitCallbacks = nil
	exitMu.Unlock()
	for _, fn := range cbs {
		fn()
	}
}

// RestoreTerminal は Alt Screen を抜けてターミナルを復旧する。
func RestoreTerminal() {
	fmt.Fprint(os.Stdout, "\033[?1049l\033[?25h")
}
