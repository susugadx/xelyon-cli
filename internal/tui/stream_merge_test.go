package tui

import "testing"

func TestApplyANSICode_Reset(t *testing.T) {
	if got := applyANSICode("\033[31m", "\033[0m"); got != "" {
		t.Fatalf("applyANSICode(reset) = %q, want empty", got)
	}
}

func TestParseStreamCells_PreservesStyledClusters(t *testing.T) {
	cells := parseStreamCells("\033[31m赤A\033[0m")
	if len(cells) == 0 {
		t.Fatal("cells should not be empty")
	}
	if cells[0].style == "" {
		t.Fatal("first cell should preserve ANSI style")
	}
	if cells[0].span <= 0 {
		t.Fatalf("first cell span = %d, want > 0", cells[0].span)
	}
}

func TestMergeStreamFragment_PartialANSISequence(t *testing.T) {
	line, cursor, activeANSI, pending := mergeStreamFragment("", "\033[31", 0, "", "")
	if line != "" {
		t.Fatalf("line = %q, want empty while ANSI is pending", line)
	}
	if cursor != 0 {
		t.Fatalf("cursor = %d, want 0", cursor)
	}
	if activeANSI != "" {
		t.Fatalf("activeANSI = %q, want empty", activeANSI)
	}
	if pending != "\033[31" {
		t.Fatalf("pending = %q, want partial ANSI", pending)
	}

	line, cursor, activeANSI, pending = mergeStreamFragment(line, "mA", cursor, activeANSI, pending)
	if stripANSI(line) != "A" {
		t.Fatalf("line = %q, want styled A", line)
	}
	if cursor != 1 {
		t.Fatalf("cursor = %d, want 1", cursor)
	}
	if activeANSI == "" {
		t.Fatal("activeANSI should keep the applied style")
	}
	if pending != "" {
		t.Fatalf("pending = %q, want empty", pending)
	}
}

func TestMergeStreamFragment_CarriageReturnOverwritesFromColumnZero(t *testing.T) {
	line, cursor, activeANSI, pending := mergeStreamFragment("hello", "\rYo", 5, "", "")
	if stripANSI(line) != "Yollo" {
		t.Fatalf("line = %q, want %q", stripANSI(line), "Yollo")
	}
	if cursor != 2 {
		t.Fatalf("cursor = %d, want 2", cursor)
	}
	if activeANSI != "" {
		t.Fatalf("activeANSI = %q, want empty", activeANSI)
	}
	if pending != "" {
		t.Fatalf("pending = %q, want empty", pending)
	}
}
