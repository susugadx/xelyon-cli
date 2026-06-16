package lifecycle

import (
	"reflect"
	"testing"
)

func TestRunExitCallbacksRunsInRegistrationOrderAndClears(t *testing.T) {
	restoreExitCallbacks(t)

	var calls []string
	OnExit(func() {
		calls = append(calls, "first")
	})
	OnExit(func() {
		calls = append(calls, "second")
	})

	RunExitCallbacks()

	want := []string{"first", "second"}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls after first run = %#v, want %#v", calls, want)
	}

	RunExitCallbacks()
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls after second run = %#v, want unchanged %#v", calls, want)
	}
}

func restoreExitCallbacks(t *testing.T) {
	t.Helper()

	exitMu.Lock()
	original := append([]func(){}, exitCallbacks...)
	exitCallbacks = nil
	exitMu.Unlock()

	t.Cleanup(func() {
		exitMu.Lock()
		exitCallbacks = original
		exitMu.Unlock()
	})
}
