package ui

import (
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

func TestGlobalReader(t *testing.T) {
	// Initially nil
	if GetGlobalReader() != nil {
		t.Error("Expected global reader to be nil initially")
	}

	// Set and get
	reader := NewMultilineReader(nil)
	SetGlobalReader(reader)

	if GetGlobalReader() != reader {
		t.Error("Expected global reader to be set correctly")
	}

	// Cleanup
	SetGlobalReader(nil)
}
