package agent

import (
	"bytes"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/ui"
)

func newREPLInterruptTestAgent(out *bytes.Buffer) *Agent {
	return &Agent{
		Runtime: &AgentRuntime{
			UI: ui.NewRuntime(strings.NewReader(""), out, out),
		},
	}
}

func TestHandleREPLReadError_FirstInterruptContinues(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	agent := newREPLInterruptTestAgent(&out)
	var lastInterrupt time.Time

	if !handleREPLReadError(agent, ui.ErrInterrupted, &lastInterrupt) {
		t.Fatal("handleREPLReadError() = false, want continue")
	}
	if lastInterrupt.IsZero() {
		t.Fatal("lastInterrupt should be updated on first interrupt")
	}
	if !strings.Contains(out.String(), "Press Ctrl+C again within 3 seconds to exit") {
		t.Fatalf("output = %q, want interrupt guidance", out.String())
	}
}

func TestHandleREPLReadError_SecondInterruptCleansUpAndExits(t *testing.T) {
	disableColors(t)

	var out bytes.Buffer
	agent := newREPLInterruptTestAgent(&out)

	var cleanupCount atomic.Int32
	cleanupHook = func() { cleanupCount.Add(1) }
	defer func() { cleanupHook = nil }()

	originalExit := exitProcess
	defer func() { exitProcess = originalExit }()

	var exitCode atomic.Int32
	exitProcess = func(code int) {
		exitCode.Store(int32(code))
	}

	lastInterrupt := time.Now()
	if !handleREPLReadError(agent, ui.ErrInterrupted, &lastInterrupt) {
		t.Fatal("handleREPLReadError() = false, want continue")
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

func TestHandleREPLReadError_NonInterruptBreaksLoop(t *testing.T) {
	var out bytes.Buffer
	agent := newREPLInterruptTestAgent(&out)
	lastInterrupt := time.Now()

	if handleREPLReadError(agent, errors.New("boom"), &lastInterrupt) {
		t.Fatal("handleREPLReadError() = true, want false")
	}
	if out.Len() != 0 {
		t.Fatalf("output = %q, want empty", out.String())
	}
}
