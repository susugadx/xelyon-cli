package agent

import (
	"fmt"
	"os"
	"time"
)

func handleSignalInterrupt(agent *Agent, lastInterrupt *time.Time, sig os.Signal) {
	now := time.Now()
	if lastInterrupt != nil && now.Sub(*lastInterrupt) < 3*time.Second {
		if agent.exitHook != nil {
			agent.exitHook()
		}
		_, _ = fmt.Fprintln(agent.output(), "\n\n👋 Gracefully shutting down...")
		agent.Cleanup()
		exitProcess(0)
		return
	}

	if lastInterrupt != nil {
		*lastInterrupt = now
	}
	_, _ = fmt.Fprintln(agent.output(), "\n\n⚠️  Interrupted. Press Ctrl+C again within 3 seconds to exit.")
	agent.cancelActiveRequest(fmt.Sprintf("signal: %s", sig))
}
