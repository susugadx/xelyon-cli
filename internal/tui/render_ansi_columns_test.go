package tui

import "testing"

func TestDisplayColumnHelpers_HandleTabs(t *testing.T) {
	line := "a\tb"

	if got := displayColToRuneIndex(line, 1); got != 1 {
		t.Fatalf("displayColToRuneIndex(%q, 1) = %d, want 1", line, got)
	}
	if got := displayColToRuneIndexAfter(line, 1); got != 2 {
		t.Fatalf("displayColToRuneIndexAfter(%q, 1) = %d, want 2", line, got)
	}
	if got := runeIndexToDisplayCol(line, 2); got != 5 {
		t.Fatalf("runeIndexToDisplayCol(%q, 2) = %d, want 5", line, got)
	}
}

func TestDisplayColumnHelpers_HandleCombiningClusters(t *testing.T) {
	line := "e\u0301x"

	if got := displayColToRuneIndex(line, 0); got != 0 {
		t.Fatalf("displayColToRuneIndex(%q, 0) = %d, want 0", line, got)
	}
	if got := displayColToRuneIndexAfter(line, 0); got != 2 {
		t.Fatalf("displayColToRuneIndexAfter(%q, 0) = %d, want 2", line, got)
	}
	if got := runeIndexToDisplayCol(line, 2); got != 1 {
		t.Fatalf("runeIndexToDisplayCol(%q, 2) = %d, want 1", line, got)
	}
}
