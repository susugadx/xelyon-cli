package mcpsurface

import "testing"

func TestIncludeSnippetDoesNotHTMLEscapePlaceholder(t *testing.T) {
	got := IncludeSnippet(Recommendation{ServerName: "beta"})
	want := `"beta": {"tools": {"include": ["<needed_tool>"]}}`
	if got != want {
		t.Fatalf("IncludeSnippet() = %q, want %q", got, want)
	}
}

func TestFormatReasonCountsAppliesLimit(t *testing.T) {
	got := FormatReasonCounts([]ReasonCount{
		{Reason: "token_budget_exceeded", Count: 2},
		{Reason: "schema_too_large", Count: 1},
	}, 1)
	want := "token_budget_exceeded=2, +1 more"
	if got != want {
		t.Fatalf("FormatReasonCounts() = %q, want %q", got, want)
	}
}
