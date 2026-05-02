package review

import "testing"

func TestNewCurrentChangesRequest(t *testing.T) {
	got := NewCurrentChangesRequest("focus on correctness")
	if got.TargetKind != TargetCurrentChanges {
		t.Fatalf("TargetKind = %q, want %q", got.TargetKind, TargetCurrentChanges)
	}
	if got.CustomInstructions != "focus on correctness" {
		t.Fatalf("CustomInstructions = %q", got.CustomInstructions)
	}
}
