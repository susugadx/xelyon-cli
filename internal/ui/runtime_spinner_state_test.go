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
