package api

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

type errAfterLineReader struct {
	sent bool
}

func (r *errAfterLineReader) Read(p []byte) (int, error) {
	if !r.sent {
		r.sent = true
		line := "line-1\n"
		copy(p, line)
		return len(line), nil
	}
	return 0, errors.New("scan boom")
}

func TestStreamLoopController_ContextDone(t *testing.T) {
	pr, pw := io.Pipe()
	defer pw.Close()

	controller := NewStreamLoopController(pr, StreamLoopOptions{})
	defer controller.Stop()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	event := controller.Next(ctx, nil)
	if event.Type != StreamLoopEventContextDone {
		t.Fatalf("event.Type = %v, want %v", event.Type, StreamLoopEventContextDone)
	}
	if !errors.Is(event.Err, context.Canceled) {
		t.Fatalf("event.Err = %v, want context canceled", event.Err)
	}
}

func TestStreamLoopController_IdleTimeout(t *testing.T) {
	pr, pw := io.Pipe()
	defer pw.Close()

	timeout := 80 * time.Millisecond
	controller := NewStreamLoopController(pr, StreamLoopOptions{
		IdleTimeout: timeout,
	})
	defer controller.Stop()

	event := controller.Next(context.Background(), nil)
	if event.Type != StreamLoopEventIdleTimeout {
		t.Fatalf("event.Type = %v, want %v", event.Type, StreamLoopEventIdleTimeout)
	}
	if event.IdleTimeout != timeout {
		t.Fatalf("event.IdleTimeout = %v, want %v", event.IdleTimeout, timeout)
	}
}

func TestStreamLoopController_ExternalEvent(t *testing.T) {
	pr, pw := io.Pipe()
	defer pw.Close()

	controller := NewStreamLoopController(pr, StreamLoopOptions{
		IdleTimeout: 500 * time.Millisecond,
	})
	defer controller.Stop()

	externalCh := make(chan time.Time, 1)
	go func() {
		time.Sleep(30 * time.Millisecond)
		externalCh <- time.Now()
	}()

	event := controller.Next(context.Background(), externalCh)
	if event.Type != StreamLoopEventExternal {
		t.Fatalf("event.Type = %v, want %v", event.Type, StreamLoopEventExternal)
	}
}

func TestStreamLoopController_ResetIdleTimer(t *testing.T) {
	pr, pw := io.Pipe()
	defer pw.Close()

	controller := NewStreamLoopController(pr, StreamLoopOptions{
		IdleTimeout: 200 * time.Millisecond,
	})
	defer controller.Stop()

	externalCh := make(chan time.Time, 1)
	go func() {
		time.Sleep(120 * time.Millisecond)
		controller.ResetIdleTimer()
		time.Sleep(120 * time.Millisecond)
		externalCh <- time.Now()
	}()

	event := controller.Next(context.Background(), externalCh)
	if event.Type != StreamLoopEventExternal {
		t.Fatalf("event.Type = %v, want %v", event.Type, StreamLoopEventExternal)
	}
}

func TestStreamLoopController_ScannerDoneError(t *testing.T) {
	controller := NewStreamLoopController(&errAfterLineReader{}, StreamLoopOptions{
		AutoResetIdleOnLine: true,
	})
	defer controller.Stop()

	event := controller.Next(context.Background(), nil)
	if event.Type != StreamLoopEventLine || strings.TrimSpace(event.Line) != "line-1" {
		t.Fatalf("first event = %+v, want line event", event)
	}

	event = controller.Next(context.Background(), nil)
	if event.Type != StreamLoopEventScannerDone {
		t.Fatalf("event.Type = %v, want %v", event.Type, StreamLoopEventScannerDone)
	}
	if event.Err == nil || !strings.Contains(event.Err.Error(), "scan boom") {
		t.Fatalf("event.Err = %v, want scan boom", event.Err)
	}
}
