package termtext

import "testing"

func TestDisplayColumnHelpers_HandleTabs(t *testing.T) {
	line := "a\tb"

	if got := DisplayColToRuneIndex(line, 1); got != 1 {
		t.Fatalf("DisplayColToRuneIndex(%q, 1) = %d, want 1", line, got)
	}
	if got := DisplayColToRuneIndexAfter(line, 1); got != 2 {
		t.Fatalf("DisplayColToRuneIndexAfter(%q, 1) = %d, want 2", line, got)
	}
	if got := RuneIndexToDisplayCol(line, 2); got != 5 {
		t.Fatalf("RuneIndexToDisplayCol(%q, 2) = %d, want 5", line, got)
	}
}

func TestDisplayColumnHelpers_HandleCombiningClusters(t *testing.T) {
	line := "e\u0301x"

	if got := DisplayColToRuneIndex(line, 0); got != 0 {
		t.Fatalf("DisplayColToRuneIndex(%q, 0) = %d, want 0", line, got)
	}
	if got := DisplayColToRuneIndexAfter(line, 0); got != 2 {
		t.Fatalf("DisplayColToRuneIndexAfter(%q, 0) = %d, want 2", line, got)
	}
	if got := RuneIndexToDisplayCol(line, 2); got != 1 {
		t.Fatalf("RuneIndexToDisplayCol(%q, 2) = %d, want 1", line, got)
	}
}
