package tools

import "testing"

func TestJSONToolCallScanState(t *testing.T) {
	state := newJSONToolCallScanState()

	if state.IsDone() {
		t.Fatalf("newJSONToolCallScanState().IsDone() = true, want false")
	}
	if got := state.SearchFrom(); got != 0 {
		t.Fatalf("newJSONToolCallScanState().SearchFrom() = %d, want 0", got)
	}

	state.AdvancePast(10)
	if got := state.SearchFrom(); got != 11 {
		t.Fatalf("AdvancePast(10) => SearchFrom() = %d, want 11", got)
	}

	state.AdvanceTo(42)
	if got := state.SearchFrom(); got != 42 {
		t.Fatalf("AdvanceTo(42) => SearchFrom() = %d, want 42", got)
	}

	state.MarkDone()
	if !state.IsDone() {
		t.Fatalf("MarkDone() => IsDone() = false, want true")
	}
}
