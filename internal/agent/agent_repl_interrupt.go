package agent

import (
	"fmt"
	"time"

	"github.com/susugadx/xelyon-cli/internal/ui"
)

// handleREPLReadError は REPL の入力読み取りエラー方針を処理する。
// true を返した場合、呼び出し元は REPL を継続する。
func handleREPLReadError(agent *Agent, err error, lastInterrupt *time.Time) bool {
	if err != ui.ErrInterrupted {
		return false
	}

	now := time.Now()
	if lastInterrupt != nil && now.Sub(*lastInterrupt) < 3*time.Second {
		_, _ = fmt.Fprintln(agent.output(), "\n👋 Gracefully shutting down...")
		agent.Cleanup()
		exitProcess(0)
		return true
	}

	if lastInterrupt != nil {
		*lastInterrupt = now
	}
	_, _ = fmt.Fprintln(agent.output(), "⚠️  Interrupted. Press Ctrl+C again within 3 seconds to exit.")
	return true
}
