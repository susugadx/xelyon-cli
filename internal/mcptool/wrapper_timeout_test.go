package mcptool

import (
	"context"
	"errors"
	"github.com/susugadx/xelyon-cli/internal/mcpapproval"
	"strings"
	"testing"
	"time"
)

func TestWrapperRunUsesParentCancellationBeforeWrapperTimeout(t *testing.T) {
	caller := &contextWaitingCaller{}
	wrapper := NewWrapper(WrapperOptions{
		Caller:      caller,
		ServerName:  "github",
		ToolName:    "slow",
		CallTimeout: time.Second,
		Approval:    mcpapproval.ModeAuto,
	})
	parentCtx, cancel := context.WithCancel(context.Background())
	cancel()

	started := time.Now()
	result, _, err := wrapper.Run(newAutoApprovedExecutionContext(parentCtx), map[string]string{})
	elapsed := time.Since(started)
	if err == nil || !errors.Is(err, context.Canceled) {
		t.Fatalf("Run() error = %v, want context.Canceled", err)
	}
	if !strings.Contains(result, "request context") {
		t.Fatalf("result = %q, want request context cancellation message", result)
	}
	if caller.calls != 1 {
		t.Fatalf("caller calls = %d, want 1", caller.calls)
	}
	if elapsed >= 200*time.Millisecond {
		t.Fatalf("Run() elapsed = %v, want parent cancellation before wrapper timeout", elapsed)
	}
}

func TestWrapperRunUsesParentDeadlineBeforeWrapperTimeout(t *testing.T) {
	caller := &contextWaitingCaller{}
	wrapper := NewWrapper(WrapperOptions{
		Caller:      caller,
		ServerName:  "github",
		ToolName:    "slow",
		CallTimeout: time.Second,
		Approval:    mcpapproval.ModeAuto,
	})
	parentCtx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	started := time.Now()
	result, _, err := wrapper.Run(newAutoApprovedExecutionContext(parentCtx), map[string]string{})
	elapsed := time.Since(started)
	if err == nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Run() error = %v, want context.DeadlineExceeded", err)
	}
	if !strings.Contains(result, "request deadline") {
		t.Fatalf("result = %q, want request deadline message", result)
	}
	if caller.calls != 1 {
		t.Fatalf("caller calls = %d, want 1", caller.calls)
	}
	if elapsed >= 500*time.Millisecond {
		t.Fatalf("Run() elapsed = %v, want parent deadline before wrapper timeout", elapsed)
	}
}

func TestWrapperRunUsesWrapperTimeoutWhenParentStillActive(t *testing.T) {
	caller := &contextWaitingCaller{}
	wrapper := NewWrapper(WrapperOptions{
		Caller:      caller,
		ServerName:  "github",
		ToolName:    "slow",
		CallTimeout: 20 * time.Millisecond,
		Approval:    mcpapproval.ModeAuto,
	})

	started := time.Now()
	result, _, err := wrapper.Run(newAutoApprovedExecutionContext(context.Background()), map[string]string{})
	elapsed := time.Since(started)
	if err == nil || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Run() error = %v, want context.DeadlineExceeded", err)
	}
	if !strings.Contains(result, "timed out after 20ms") {
		t.Fatalf("result = %q, want wrapper timeout message", result)
	}
	if caller.calls != 1 {
		t.Fatalf("caller calls = %d, want 1", caller.calls)
	}
	if elapsed >= 500*time.Millisecond {
		t.Fatalf("Run() elapsed = %v, want wrapper timeout before long wait", elapsed)
	}
}
