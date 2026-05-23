package navigation

import "testing"

func TestRipgrepReferenceSearchArgsUsesProvidedSearchPath(t *testing.T) {
	args := ripgrepReferenceSearchArgs("Target", "pkg/app")
	if got := args[len(args)-1]; got != "pkg/app" {
		t.Fatalf("search path arg = %q, want pkg/app", got)
	}
}

func TestRipgrepReferenceSearchArgsDefaultsToDot(t *testing.T) {
	args := ripgrepReferenceSearchArgs("Target", " ")
	if got := args[len(args)-1]; got != "." {
		t.Fatalf("search path arg = %q, want .", got)
	}
}
