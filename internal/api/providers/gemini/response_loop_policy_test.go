package gemini

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestSSELoopPolicy_ResolveContextDoneWithPartial(t *testing.T) {
	policy := newSSELoopPolicy(context.Background(), 30*time.Second)

	transition := policy.resolveContextDone("partial", errors.New("event err"))
	if !transition.terminate {
		t.Fatal("resolveContextDone() terminate = false, want true")
	}
	if transition.response != "partial" {
		t.Fatalf("resolveContextDone() response = %q, want %q", transition.response, "partial")
	}
	if transition.err != nil {
		t.Fatalf("resolveContextDone() err = %v, want nil", transition.err)
	}
}

func TestSSELoopPolicy_ResolveContextDonePrefersEventErr(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	eventErr := errors.New("stream loop context done")
	policy := newSSELoopPolicy(ctx, 30*time.Second)

	transition := policy.resolveContextDone("", eventErr)
	if !transition.terminate {
		t.Fatal("resolveContextDone() terminate = false, want true")
	}
	if !errors.Is(transition.err, eventErr) {
		t.Fatalf("resolveContextDone() err = %v, want %v", transition.err, eventErr)
	}
}

func TestSSELoopPolicy_ResolveContextDoneFallsBackToContextErr(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	policy := newSSELoopPolicy(ctx, 30*time.Second)

	transition := policy.resolveContextDone("", nil)
	if !transition.terminate {
		t.Fatal("resolveContextDone() terminate = false, want true")
	}
	if !errors.Is(transition.err, context.Canceled) {
		t.Fatalf("resolveContextDone() err = %v, want %v", transition.err, context.Canceled)
	}
}

func TestSSELoopPolicy_ResolveIdleTimeout(t *testing.T) {
	policy := newSSELoopPolicy(context.Background(), 42*time.Second)

	transition := policy.resolveIdleTimeout("partial")
	if !transition.terminate {
		t.Fatal("resolveIdleTimeout() terminate = false, want true")
	}
	if transition.response != "partial" {
		t.Fatalf("resolveIdleTimeout() response = %q, want %q", transition.response, "partial")
	}
	if transition.err == nil {
		t.Fatal("resolveIdleTimeout() err = nil, want timeout error")
	}
	if !strings.Contains(transition.err.Error(), "transport idle timeout") {
		t.Fatalf("resolveIdleTimeout() err = %q, want idle timeout message", transition.err.Error())
	}
}

func TestSSELoopPolicy_ResolveScannerDoneWithError(t *testing.T) {
	policy := newSSELoopPolicy(context.Background(), 30*time.Second)
	scanErr := errors.New("boom")

	transition := policy.resolveScannerDone(scanErr)
	if !transition.terminate {
		t.Fatal("resolveScannerDone() terminate = false, want true")
	}
	if transition.err == nil || !strings.Contains(transition.err.Error(), "SSE scan error: boom") {
		t.Fatalf("resolveScannerDone() err = %v, want wrapped scan error", transition.err)
	}
}

func TestSSELoopPolicy_ResolveScannerDoneWithoutError(t *testing.T) {
	policy := newSSELoopPolicy(context.Background(), 30*time.Second)

	transition := policy.resolveScannerDone(nil)
	if transition.terminate {
		t.Fatal("resolveScannerDone() terminate = true, want false")
	}
	if !transition.continueToDone {
		t.Fatal("resolveScannerDone() continueToDone = false, want true")
	}
	if transition.err != nil {
		t.Fatalf("resolveScannerDone() err = %v, want nil", transition.err)
	}
}
