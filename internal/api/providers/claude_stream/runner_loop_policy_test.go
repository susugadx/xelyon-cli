package claudestream

import (
	"bytes"
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/api"
	"github.com/susugadx/xelyon-cli/internal/ui"
)

func newRunnerLoopPolicyTestState() *runnerOutputState {
	out := &bytes.Buffer{}
	errOut := &bytes.Buffer{}
	ctx := ui.WithRuntime(context.Background(), ui.NewRuntime(strings.NewReader(""), out, errOut))
	ctx = api.WithAssistantUpdateMode(ctx, api.AssistantUpdatesPhase)
	return newRunnerOutputState(ctx, nil)
}

func TestRunnerLoopPolicy_ResolveContextDone(t *testing.T) {
	state := newRunnerLoopPolicyTestState()
	policy := newRunnerLoopPolicy(state, nil, RunnerOptions{}, time.Second)
	eventErr := errors.New("context done")

	transition := policy.resolve(api.StreamLoopEvent{
		Type: api.StreamLoopEventContextDone,
		Err:  eventErr,
	})
	if !transition.handled {
		t.Fatal("resolve() handled = false, want true")
	}
	if !errors.Is(transition.err, eventErr) {
		t.Fatalf("resolve() err = %v, want %v", transition.err, eventErr)
	}
	if transition.response != "" {
		t.Fatalf("resolve() response = %q, want empty", transition.response)
	}
}

func TestRunnerLoopPolicy_ResolveLineDone(t *testing.T) {
	state := newRunnerLoopPolicyTestState()
	handler := func(_ StreamEvent, _ string) (string, bool, error) {
		return "", true, nil
	}
	policy := newRunnerLoopPolicy(state, handler, RunnerOptions{}, time.Second)

	transition := policy.resolve(api.StreamLoopEvent{
		Type: api.StreamLoopEventLine,
		Line: `data: {"type":"message_stop"}`,
	})
	if !transition.handled {
		t.Fatal("resolve() handled = false, want true")
	}
	if transition.err != nil {
		t.Fatalf("resolve() err = %v, want nil", transition.err)
	}
}

func TestRunnerLoopPolicy_ResolveLineTextDeltaAppendsState(t *testing.T) {
	state := newRunnerLoopPolicyTestState()
	handler := func(_ StreamEvent, _ string) (string, bool, error) {
		return "hello", false, nil
	}
	policy := newRunnerLoopPolicy(state, handler, RunnerOptions{}, time.Second)

	transition := policy.resolve(api.StreamLoopEvent{
		Type: api.StreamLoopEventLine,
		Line: `data: {"type":"content_block_delta","delta":{"type":"text_delta","text":"ignored"}}`,
	})
	if transition.handled {
		t.Fatal("resolve() handled = true, want false")
	}
	if got := state.response(); got != "hello" {
		t.Fatalf("state.response() = %q, want %q", got, "hello")
	}
}

func TestRunnerLoopPolicy_ResolveLineDecodeError(t *testing.T) {
	state := newRunnerLoopPolicyTestState()
	handler := func(_ StreamEvent, _ string) (string, bool, error) {
		return "", false, nil
	}
	policy := newRunnerLoopPolicy(state, handler, RunnerOptions{IgnoreDecodeError: false}, time.Second)

	transition := policy.resolve(api.StreamLoopEvent{
		Type: api.StreamLoopEventLine,
		Line: `data: {broken json`,
	})
	if !transition.handled {
		t.Fatal("resolve() handled = false, want true")
	}
	if transition.err == nil {
		t.Fatal("resolve() err = nil, want decode error")
	}
}

func TestRunnerLoopPolicy_ResolveLineDecodeErrorIgnored(t *testing.T) {
	state := newRunnerLoopPolicyTestState()
	handler := func(_ StreamEvent, _ string) (string, bool, error) {
		return "", false, nil
	}
	policy := newRunnerLoopPolicy(state, handler, RunnerOptions{IgnoreDecodeError: true}, time.Second)

	transition := policy.resolve(api.StreamLoopEvent{
		Type: api.StreamLoopEventLine,
		Line: `data: {broken json`,
	})
	if transition.handled {
		t.Fatal("resolve() handled = true, want false")
	}
}

func TestRunnerLoopPolicy_ResolveIdleTimeout(t *testing.T) {
	state := newRunnerLoopPolicyTestState()
	policy := newRunnerLoopPolicy(state, nil, RunnerOptions{}, 7*time.Second)

	transition := policy.resolve(api.StreamLoopEvent{Type: api.StreamLoopEventIdleTimeout})
	if !transition.handled {
		t.Fatal("resolve() handled = false, want true")
	}
	if transition.err == nil || !strings.Contains(transition.err.Error(), "idle timeout: no data received for 7s") {
		t.Fatalf("resolve() err = %v, want idle timeout error", transition.err)
	}
}

func TestRunnerLoopPolicy_ResolveScannerDoneWithError(t *testing.T) {
	state := newRunnerLoopPolicyTestState()
	policy := newRunnerLoopPolicy(state, nil, RunnerOptions{}, time.Second)

	transition := policy.resolve(api.StreamLoopEvent{
		Type: api.StreamLoopEventScannerDone,
		Err:  errors.New("scan failed"),
	})
	if !transition.handled {
		t.Fatal("resolve() handled = false, want true")
	}
	if transition.err == nil || !strings.Contains(transition.err.Error(), "stream reading error: scan failed") {
		t.Fatalf("resolve() err = %v, want scanner error", transition.err)
	}
}
