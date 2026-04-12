package ui

import (
	"bytes"
	"strings"
	"testing"
)

func TestSelectorRunWithIO_InputVariants(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "default enter", input: "\n", want: "yes"},
		{name: "no shortcut", input: "n\n", want: "no"},
		{name: "comment numeric", input: "3\n", want: "comment"},
		{name: "comment literal", input: "comment\n", want: "comment"},
		{name: "explicit second option", input: "2\n", want: "no"},
		{name: "unknown falls back to yes", input: "invalid\n", want: "yes"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var out bytes.Buffer
			selector := NewSelector("Proceed", []SelectOption{
				{Label: "Yes", Description: "Apply", Value: "yes"},
				{Label: "No", Description: "Skip", Value: "no"},
				{Label: "Comment", Description: "Comment", Value: "comment"},
			})
			got, err := selector.RunWithIO(NewPromptIO(strings.NewReader(tt.input), &out, &out, nil))
			if err != nil {
				t.Fatalf("RunWithIO() error = %v", err)
			}
			if got != tt.want {
				t.Fatalf("RunWithIO() = %q, want %q", got, tt.want)
			}
			if !strings.Contains(out.String(), "Choice [1]:") {
				t.Fatalf("expected prompt output, got %q", out.String())
			}
		})
	}
}

func TestConfirmSelectorWithIO_UsesThreeWaySelector(t *testing.T) {
	var out bytes.Buffer
	got, err := ConfirmSelectorWithIO(NewPromptIO(strings.NewReader("c\n"), &out, &out, nil), "continue?")
	if err != nil {
		t.Fatalf("ConfirmSelectorWithIO() error = %v", err)
	}
	if got != "comment" {
		t.Fatalf("ConfirmSelectorWithIO() = %q, want %q", got, "comment")
	}
	if !strings.Contains(out.String(), "continue?") {
		t.Fatalf("expected selector message in output, got %q", out.String())
	}
}
