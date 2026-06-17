package ui

import (
	"bytes"
	"testing"
)

func TestRuntime_CurrentSpinnerClearsInactiveSpinner(t *testing.T) {
	runtime := NewRuntime(nil, &bytes.Buffer{}, &bytes.Buffer{})
	spinner := runtime.NewSpinner()

	runtime.SetSpinner(spinner)
	spinner.Start("Testing")
	spinner.Stop()

	if got := runtime.CurrentSpinner(); got != nil {
		t.Fatal("CurrentSpinner() should clear and return nil for inactive spinner")
	}
}

func TestRuntime_CurrentSpinnerKeepsUnstartedSpinner(t *testing.T) {
	runtime := NewRuntime(nil, &bytes.Buffer{}, &bytes.Buffer{})
	spinner := runtime.NewSpinner()

	runtime.SetSpinner(spinner)

	if got := runtime.CurrentSpinner(); got != spinner {
		t.Fatal("CurrentSpinner() should keep spinner that has not started yet")
	}
}

func TestRuntime_StartSpinnerRegistersCurrentSpinner(t *testing.T) {
	runtime := NewRuntime(nil, &bytes.Buffer{}, &bytes.Buffer{})

	spinner := runtime.StartSpinner("Testing")
	if spinner == nil {
		t.Fatal("StartSpinner() returned nil")
	}
	if !spinner.IsActive() {
		t.Fatal("StartSpinner() should start the spinner")
	}
	if got := runtime.CurrentSpinner(); got != spinner {
		t.Fatal("StartSpinner() should register the spinner as current")
	}

	runtime.StopSpinner()
	if got := runtime.CurrentSpinner(); got != nil {
		t.Fatal("StopSpinner() should clear the current spinner")
	}
}

func TestRuntime_StartSpinnerStopsPreviousSpinner(t *testing.T) {
	runtime := NewRuntime(nil, &bytes.Buffer{}, &bytes.Buffer{})

	first := runtime.StartSpinner("First")
	second := runtime.StartSpinner("Second")

	if first.IsActive() {
		t.Fatal("StartSpinner() should stop the previous current spinner")
	}
	if second == nil || !second.IsActive() {
		t.Fatal("StartSpinner() should start the replacement spinner")
	}
	if got := runtime.CurrentSpinner(); got != second {
		t.Fatal("StartSpinner() should replace the current spinner")
	}

	runtime.StopSpinner()
}
