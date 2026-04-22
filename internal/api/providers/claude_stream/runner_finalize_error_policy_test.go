package claudestream

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestRunnerFinalizeErrorPolicy_ResolveIdleTimeout(t *testing.T) {
	policy := newRunnerFinalizeErrorPolicy()

	resolution := policy.resolveIdleTimeout(42 * time.Second)
	if resolution.err == nil {
		t.Fatal("resolveIdleTimeout() err = nil, want timeout error")
	}
	if !strings.Contains(resolution.err.Error(), "idle timeout: no data received for 42s") {
		t.Fatalf("resolveIdleTimeout() err = %q, want idle timeout message", resolution.err.Error())
	}
	if resolution.printTrailingNewline {
		t.Fatal("resolveIdleTimeout() printTrailingNewline = true, want false")
	}
}

func TestRunnerFinalizeErrorPolicy_ResolveScannerDoneWithError(t *testing.T) {
	policy := newRunnerFinalizeErrorPolicy()
	scanErr := errors.New("broken pipe")

	resolution := policy.resolveScannerDone(scanErr)
	if resolution.err == nil {
		t.Fatal("resolveScannerDone() err = nil, want wrapped scanner error")
	}
	if !strings.Contains(resolution.err.Error(), "stream reading error: broken pipe") {
		t.Fatalf("resolveScannerDone() err = %q, want wrapped scanner error message", resolution.err.Error())
	}
	if resolution.printTrailingNewline {
		t.Fatal("resolveScannerDone() printTrailingNewline = true, want false on error")
	}
}

func TestRunnerFinalizeErrorPolicy_ResolveScannerDoneWithoutError(t *testing.T) {
	policy := newRunnerFinalizeErrorPolicy()

	resolution := policy.resolveScannerDone(nil)
	if resolution.err != nil {
		t.Fatalf("resolveScannerDone() err = %v, want nil", resolution.err)
	}
	if !resolution.printTrailingNewline {
		t.Fatal("resolveScannerDone() printTrailingNewline = false, want true")
	}
}

func TestRunnerFinalizeErrorPolicy_ResolveDecodeError(t *testing.T) {
	policy := newRunnerFinalizeErrorPolicy()
	decodeErr := errors.New("invalid json")

	resolution := policy.resolveDecodeError(decodeErr)
	if !errors.Is(resolution.err, decodeErr) {
		t.Fatalf("resolveDecodeError() err = %v, want %v", resolution.err, decodeErr)
	}
	if !resolution.printTrailingNewline {
		t.Fatal("resolveDecodeError() printTrailingNewline = false, want true")
	}
}

func TestRunnerFinalizeErrorPolicy_ResolveHandlerError(t *testing.T) {
	policy := newRunnerFinalizeErrorPolicy()
	handlerErr := errors.New("handler failed")

	resolution := policy.resolveHandlerError(handlerErr)
	if !errors.Is(resolution.err, handlerErr) {
		t.Fatalf("resolveHandlerError() err = %v, want %v", resolution.err, handlerErr)
	}
	if !resolution.printTrailingNewline {
		t.Fatal("resolveHandlerError() printTrailingNewline = false, want true")
	}
}

func TestRunnerFinalizeErrorPolicy_ResolveDone(t *testing.T) {
	policy := newRunnerFinalizeErrorPolicy()

	resolution := policy.resolveDone()
	if resolution.err != nil {
		t.Fatalf("resolveDone() err = %v, want nil", resolution.err)
	}
	if !resolution.printTrailingNewline {
		t.Fatal("resolveDone() printTrailingNewline = false, want true")
	}
}
