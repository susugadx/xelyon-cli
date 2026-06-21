package agent

import (
	"bytes"
	"os"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/uiruntime"
)

func newSignalInterruptTestAgent(out *bytes.Buffer) *Agent {
	return &Agent{
		Runtime: &AgentRuntime{
			UI: uiruntime.NewRuntime(strings.NewReader(""), out, out),
		},
	}
}

func TestHandleSignalInterrupt_FirstSignalCancelsActiveRequest(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	agent := newSignalInterruptTestAgent(&out)
	var canceled atomic.Int32
	agent.cancelFunc = func() { canceled.Add(1) }

	var lastInterrupt time.Time
	handleSignalInterrupt(agent, &lastInterrupt, os.Interrupt)

	if lastInterrupt.IsZero() {
		t.Fatal("lastInterrupt should be updated")
	}
	if canceled.Load() != 1 {
		t.Fatalf("cancel count = %d, want 1", canceled.Load())
	}
	if agent.lastCancelReason != "signal: interrupt" {
		t.Fatalf("lastCancelReason = %q, want %q", agent.lastCancelReason, "signal: interrupt")
	}
	if !strings.Contains(out.String(), "Press Ctrl+C again within 3 seconds to exit") {
		t.Fatalf("output = %q, want interrupt guidance", out.String())
	}
}

func TestHandleSignalInterrupt_SecondSignalRunsExitPath(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	agent := newSignalInterruptTestAgent(&out)

	var cleanupCount atomic.Int32
	cleanupHook = func() { cleanupCount.Add(1) }
	defer func() { cleanupHook = nil }()

	var exitHookCount atomic.Int32
	agent.exitHook = func() { exitHookCount.Add(1) }

	originalExit := exitProcess
	defer func() { exitProcess = originalExit }()

	var exitCode atomic.Int32
	exitProcess = func(code int) { exitCode.Store(int32(code)) }

	lastInterrupt := time.Now()
	handleSignalInterrupt(agent, &lastInterrupt, os.Interrupt)

	if exitHookCount.Load() != 1 {
		t.Fatalf("exitHook count = %d, want 1", exitHookCount.Load())
	}
	if cleanupCount.Load() != 1 {
		t.Fatalf("cleanup count = %d, want 1", cleanupCount.Load())
	}
	if exitCode.Load() != 0 {
		t.Fatalf("exit code = %d, want 0", exitCode.Load())
	}
	if !strings.Contains(out.String(), "Gracefully shutting down") {
		t.Fatalf("output = %q, want shutdown message", out.String())
	}
}

func TestAgentCleanupRunsSignalCleanup(t *testing.T) {
	var out bytes.Buffer
	agent := newSignalInterruptTestAgent(&out)

	var signalCleanupCount atomic.Int32
	agent.signalCleanup = func() { signalCleanupCount.Add(1) }

	agent.Cleanup()

	if signalCleanupCount.Load() != 1 {
		t.Fatalf("signal cleanup count = %d, want 1", signalCleanupCount.Load())
	}
}
