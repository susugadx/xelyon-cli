package uiruntime

import (
	"bytes"
	"strings"
	"testing"
)

func TestNewSelector(t *testing.T) {
	options := []SelectOption{
		{Label: "Yes", Description: "Confirm", Value: "yes"},
		{Label: "No", Description: "Cancel", Value: "no"},
	}

	selector := NewSelector("Test message", options)

	if selector.Message != "Test message" {
		t.Errorf("Expected message 'Test message', got '%s'", selector.Message)
	}

	if len(selector.Options) != 2 {
		t.Errorf("Expected 2 options, got %d", len(selector.Options))
	}

	if selector.Options[0].Value != "yes" {
		t.Errorf("Expected first option value 'yes', got '%s'", selector.Options[0].Value)
	}
}

func TestSelectOption(t *testing.T) {
	opt := SelectOption{
		Label:       "Test",
		Description: "Test description",
		Value:       "test",
	}

	if opt.Label != "Test" {
		t.Errorf("Expected label 'Test', got '%s'", opt.Label)
	}

	if opt.Description != "Test description" {
		t.Errorf("Expected description 'Test description', got '%s'", opt.Description)
	}

	if opt.Value != "test" {
		t.Errorf("Expected value 'test', got '%s'", opt.Value)
	}
}

func TestSelector_RunWithIO_StopsRuntimeSpinner(t *testing.T) {
	runtime := NewRuntime(strings.NewReader("1\n"), &bytes.Buffer{}, &bytes.Buffer{})
	runtime.SetSpinner(NewSpinnerWithRuntime(runtime))

	selector := NewSelector("Test message", []SelectOption{
		{Label: "Yes", Description: "Confirm", Value: "yes"},
	})

	result, err := selector.RunWithIO(runtime.PromptIO())
	if err != nil {
		t.Fatalf("RunWithIO() error = %v", err)
	}
	if result != "yes" {
		t.Fatalf("RunWithIO() = %q, want %q", result, "yes")
	}
	if runtime.CurrentSpinner() != nil {
		t.Fatal("expected runtime spinner to be cleared")
	}
	output := runtime.Output().(*bytes.Buffer).String()
	if !strings.Contains(output, "Choice [1]:") {
		t.Fatalf("expected injected output to contain selector prompt, got %q", output)
	}
}
