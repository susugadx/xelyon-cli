package uiprompt

import "testing"

func TestConfirmPromptOptions_UsesCustomOrderAndFiltersComment(t *testing.T) {
	req := PromptRequest{
		Kind:         PromptKindConfirm,
		AllowComment: false,
		Options: []PromptOption{
			{Label: "Approve", Value: string(PromptActionYes)},
			{Label: "Request changes", Value: string(PromptActionComment)},
			{Label: "Cancel", Value: string(PromptActionNo)},
			{Label: "Ignore", Value: "later"},
		},
	}

	options := ConfirmPromptOptions(req, []PromptOption{
		{Label: "Fallback yes", Value: string(PromptActionYes)},
	})

	if len(options) != 2 {
		t.Fatalf("len(options) = %d, want 2", len(options))
	}
	if options[0].Label != "Approve" || options[0].Value != string(PromptActionYes) {
		t.Fatalf("options[0] = %#v, want custom yes", options[0])
	}
	if options[1].Label != "Cancel" || options[1].Value != string(PromptActionNo) {
		t.Fatalf("options[1] = %#v, want custom no", options[1])
	}
}

func TestConfirmPromptOptions_FallsBackWhenCustomOptionsInvalid(t *testing.T) {
	req := PromptRequest{
		Kind:         PromptKindConfirm,
		AllowComment: true,
		Options: []PromptOption{
			{Label: "Invalid", Value: "later"},
		},
	}

	options := ConfirmPromptOptions(req, []PromptOption{
		{Label: "Yes", Value: string(PromptActionYes)},
		{Label: "Comment", Value: string(PromptActionComment)},
	})

	if len(options) != 2 {
		t.Fatalf("len(options) = %d, want fallback options", len(options))
	}
	if options[0].Value != string(PromptActionYes) || options[1].Value != string(PromptActionComment) {
		t.Fatalf("options = %#v, want fallback yes/comment", options)
	}
}

func TestConfirmPromptOptionMatchesInput(t *testing.T) {
	option := PromptOption{Label: "Request changes", Value: string(PromptActionComment)}

	for _, input := range []string{"request changes", "comment", "c"} {
		t.Run(input, func(t *testing.T) {
			if !ConfirmPromptOptionMatchesInput(input, option) {
				t.Fatalf("ConfirmPromptOptionMatchesInput(%q) = false, want true", input)
			}
		})
	}
}

func TestConfirmPromptActionShortcut(t *testing.T) {
	tests := []struct {
		input string
		want  PromptAction
		ok    bool
	}{
		{input: "y", want: PromptActionYes, ok: true},
		{input: "yes", want: PromptActionYes, ok: true},
		{input: "n", want: PromptActionNo, ok: true},
		{input: "comment", want: PromptActionComment, ok: true},
		{input: "x", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, ok := ConfirmPromptActionShortcut(tt.input)
			if ok != tt.ok || got != tt.want {
				t.Fatalf("ConfirmPromptActionShortcut(%q) = %q, %v; want %q, %v", tt.input, got, ok, tt.want, tt.ok)
			}
		})
	}
}
