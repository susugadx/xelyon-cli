package lifecycle

import (
	"log"
	"os"
)

// debugLog は TUI デバッグログ用のロガー（XELYON_TUI_DEBUG=1 で有効化）。
var debugLog *log.Logger

func init() {
	if os.Getenv("XELYON_TUI_DEBUG") != "1" {
		return
	}

	f, err := os.OpenFile("/tmp/xelyon-tui.log", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return
	}
	debugLog = log.New(f, "[TUI] ", log.Ltime|log.Lmicroseconds)
}

// DebugLog は TUI デバッグログに出力する。
func DebugLog(format string, args ...any) {
	if debugLog != nil {
		debugLog.Printf(format, args...)
	}
}
