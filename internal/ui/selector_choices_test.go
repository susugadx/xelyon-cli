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
		{name: "eof falls back to default", input: "", want: "yes"},
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

func TestConfirmSelectorRequestWithIO_UsesCustomConfirmOptionOrder(t *testing.T) {
	var out bytes.Buffer
	req := NewPlanApprovalPromptRequest()
	got, err := ConfirmSelectorRequestWithIO(NewPromptIO(strings.NewReader("2\n"), &out, &out, nil), req)
	if err != nil {
		t.Fatalf("ConfirmSelectorRequestWithIO() error = %v", err)
	}
	if got != string(PromptActionComment) {
		t.Fatalf("ConfirmSelectorRequestWithIO() = %q, want comment from second custom option", got)
	}
	for _, want := range []string{"Approve", "Request changes", "Cancel"} {
		if !strings.Contains(out.String(), want) {
			t.Fatalf("selector output = %q, want custom option %q", out.String(), want)
		}
	}
}

func TestConfirmSelectorRequestWithIO_UsesNamedShortcutsForCustomOptions(t *testing.T) {
	var out bytes.Buffer
	req := NewPlanApprovalPromptRequest()
	got, err := ConfirmSelectorRequestWithIO(NewPromptIO(strings.NewReader("n\n"), &out, &out, nil), req)
	if err != nil {
		t.Fatalf("ConfirmSelectorRequestWithIO() error = %v", err)
	}
	if got != string(PromptActionNo) {
		t.Fatalf("ConfirmSelectorRequestWithIO() = %q, want no from n shortcut", got)
	}
}

func TestConfirmSelectorRequestWithIO_ExplicitPolicyRePromptsOnEmptySubmit(t *testing.T) {
	var out bytes.Buffer
	req := NewPlanApprovalPromptRequest()
	got, err := ConfirmSelectorRequestWithIO(NewPromptIO(strings.NewReader("\n2\n"), &out, &out, nil), req)
	if err != nil {
		t.Fatalf("ConfirmSelectorRequestWithIO() error = %v", err)
	}
	if got != string(PromptActionComment) {
		t.Fatalf("ConfirmSelectorRequestWithIO() = %q, want comment after retry", got)
	}
	if strings.Contains(out.String(), "Enter=Approve") || !strings.Contains(out.String(), "Choice:") {
		t.Fatalf("selector output = %q, want explicit prompt without Enter default", out.String())
	}
	if !strings.Contains(out.String(), "Please choose one of the listed options.") {
		t.Fatalf("selector output = %q, want retry guidance after empty submit", out.String())
	}
}

func TestConfirmSelectorRequestWithIO_ExplicitPolicyReturnsEOFWhenNoInput(t *testing.T) {
	var out bytes.Buffer
	req := NewPlanApprovalPromptRequest()
	got, err := ConfirmSelectorRequestWithIO(NewPromptIO(strings.NewReader(""), &out, &out, nil), req)
	if err == nil {
		t.Fatal("ConfirmSelectorRequestWithIO() error = nil, want EOF")
	}
	if got != "" {
		t.Fatalf("ConfirmSelectorRequestWithIO() = %q, want empty value on EOF", got)
	}
}
