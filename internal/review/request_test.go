package review

import "testing"

func TestNewUncommittedRequest(t *testing.T) {
	got := NewUncommittedRequest("focus on correctness")
	if got.TargetKind != TargetUncommitted {
		t.Fatalf("TargetKind = %q, want %q", got.TargetKind, TargetUncommitted)
	}
	if got.CustomInstructions != "focus on correctness" {
		t.Fatalf("CustomInstructions = %q", got.CustomInstructions)
	}
}
