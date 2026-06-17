package agent

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// setupSignalHandler はシグナルハンドラーを設定する。
func setupSignalHandler(agent *Agent) func() {
	if agent == nil {
		return func() {}
	}

	sigChan := make(chan os.Signal, 1)
	done := make(chan struct{})
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	var lastInterrupt time.Time
	var interruptMu sync.Mutex
	var cleanupOnce sync.Once
	cleanup := func() {
		cleanupOnce.Do(func() {
			signal.Stop(sigChan)
			close(done)
		})
	}
	agent.signalCleanup = cleanup
	go func() {
		for {
			select {
			case sig := <-sigChan:
				interruptMu.Lock()
				handleSignalInterrupt(agent, &lastInterrupt, sig)
				interruptMu.Unlock()
			case <-done:
				return
			}
		}
	}()
	return cleanup
}
