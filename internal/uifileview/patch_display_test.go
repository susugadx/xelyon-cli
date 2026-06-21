package uifileview

import (
	"bytes"
	"strings"
	"testing"
)

func TestShowCodexStyleDiff_Added(t *testing.T) {
	f := PatchFileDisplay{
		Path:       "test.txt",
		Action:     "created",
		NewContent: "line1\nline2\nline3\n",
	}
	var buf bytes.Buffer
	ShowCodexStyleDiff(&buf, []PatchFileDisplay{f})

	out := buf.String()
	if !strings.Contains(out, "Creating test.txt (+3)") {
		t.Errorf("Expected added header, got:\n%s", out)
	}
	if !strings.Contains(out, "+ line1") {
		t.Errorf("Expected added line, got:\n%s", out)
	}
}

func TestShowCodexStyleDiff_Deleted(t *testing.T) {
	f := PatchFileDisplay{
		Path:       "test.txt",
		Action:     "deleted",
		OldContent: "line1\nline2\nline3\n",
	}
	var buf bytes.Buffer
	ShowCodexStyleDiff(&buf, []PatchFileDisplay{f})

	out := buf.String()
	if !strings.Contains(out, "Deleting test.txt (-3)") {
		t.Errorf("Expected deleted header, got:\n%s", out)
	}
}

func TestShowCodexStyleDiff_Modified(t *testing.T) {
	oldStr := "line1\nline2\nline3\nline4\nline5\n"
	newStr := "line1\nline2\nmod3\nline4\nline5\n"
	f := PatchFileDisplay{
		Path:       "test.txt",
		Action:     "modified",
		OldContent: oldStr,
		NewContent: newStr,
	}

	var buf bytes.Buffer
	ShowCodexStyleDiff(&buf, []PatchFileDisplay{f})

	out := buf.String()
	if !strings.Contains(out, "Editing test.txt (+1, -1)") {
		t.Errorf("Expected edited header, got:\n%s", out)
	}
	if !strings.Contains(out, "- line3") {
		t.Errorf("Expected removed line, got:\n%s", out)
	}
	if !strings.Contains(out, "+ mod3") {
		t.Errorf("Expected added line, got:\n%s", out)
	}
}

func TestComputeHunks_MultipleChanges(t *testing.T) {
	oldLines := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12"}
	newLines := []string{"1", "2", "mod3", "4", "5", "6", "7", "8", "9", "mod10", "11", "12"}

	hunks := computeHunks(oldLines, newLines, 1)
	if len(hunks) != 2 {
		t.Fatalf("Expected 2 hunks, got %d", len(hunks))
	}

	// hunk 1: 2, mod3, 4
	if len(hunks[0].Lines) != 4 { // ctx(2), del(3), add(mod3), ctx(4)
		t.Errorf("Expected 4 lines in hunk 1, got %d", len(hunks[0].Lines))
	}

	// hunk 2: 9, mod10, 11
	if len(hunks[1].Lines) != 4 {
		t.Errorf("Expected 4 lines in hunk 2, got %d", len(hunks[1].Lines))
	}
}

func TestComputeHunks_ContextLines(t *testing.T) {
	oldLines := []string{"1", "2", "3", "4", "5"}
	newLines := []string{"1", "2", "mod3", "4", "5"}

	hunks := computeHunks(oldLines, newLines, 2)
	if len(hunks) != 1 {
		t.Fatalf("Expected 1 hunk, got %d", len(hunks))
	}

	if len(hunks[0].Lines) != 6 { // 1, 2, del(3), add(mod3), 4, 5
		t.Errorf("Expected 6 lines in hunk, got %d", len(hunks[0].Lines))
	}
}
