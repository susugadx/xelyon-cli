package agent

import (
	"testing"
)

// --- Level 1: checkMissingWriteTools ---

func TestCheckMissingWriteTools_HasMissing(t *testing.T) {
	stepTools := []string{"read_file", "str_replace"}
	executed := map[string]bool{"read_file": true}

	missing := checkMissingWriteTools(stepTools, executed)
	if len(missing) != 1 || missing[0] != "str_replace" {
		t.Errorf("expected [str_replace], got %v", missing)
	}
}

func TestCheckMissingWriteTools_AllExecuted(t *testing.T) {
	stepTools := []string{"read_file", "str_replace", "write_file"}
	executed := map[string]bool{"read_file": true, "str_replace": true, "write_file": true}

	missing := checkMissingWriteTools(stepTools, executed)
	if len(missing) != 0 {
		t.Errorf("expected empty, got %v", missing)
	}
}

func TestCheckMissingWriteTools_ReadOnly(t *testing.T) {
	stepTools := []string{"read_file", "search_code", "bash"}
	executed := map[string]bool{"read_file": true}

	missing := checkMissingWriteTools(stepTools, executed)
	if len(missing) != 0 {
		t.Errorf("expected empty (no write tools in plan), got %v", missing)
	}
}

func TestCheckMissingWriteTools_MultipleMissing(t *testing.T) {
	stepTools := []string{"str_replace", "write_file", "delete_file"}
	executed := map[string]bool{}

	missing := checkMissingWriteTools(stepTools, executed)
	if len(missing) != 3 {
		t.Errorf("expected 3 missing, got %v", missing)
	}
}

func TestCheckMissingWriteTools_EmptyPlan(t *testing.T) {
	stepTools := []string{}
	executed := map[string]bool{"str_replace": true}

	missing := checkMissingWriteTools(stepTools, executed)
	if len(missing) != 0 {
		t.Errorf("expected empty, got %v", missing)
	}
}

// --- Level 2: diffFilesEqual ---

func TestDiffFilesEqual_Same(t *testing.T) {
	a := map[string]bool{"a.go": true, "b.go": true}
	b := map[string]bool{"a.go": true, "b.go": true}

	if !diffFilesEqual(a, b) {
		t.Error("expected true for identical sets")
	}
}

func TestDiffFilesEqual_Different(t *testing.T) {
	a := map[string]bool{"a.go": true}
	b := map[string]bool{"a.go": true, "b.go": true}

	if diffFilesEqual(a, b) {
		t.Error("expected false for different sets (different length)")
	}
}

func TestDiffFilesEqual_DifferentContent(t *testing.T) {
	a := map[string]bool{"a.go": true, "c.go": true}
	b := map[string]bool{"a.go": true, "b.go": true}

	if diffFilesEqual(a, b) {
		t.Error("expected false for different content (same length)")
	}
}

func TestDiffFilesEqual_EmptyBoth(t *testing.T) {
	a := map[string]bool{}
	b := map[string]bool{}

	if !diffFilesEqual(a, b) {
		t.Error("expected true for both empty")
	}
}

func TestDiffFilesEqual_NilBefore(t *testing.T) {
	// nil before (git not available at step start) + non-empty after
	var a map[string]bool
	b := map[string]bool{"a.go": true}

	if diffFilesEqual(a, b) {
		t.Error("expected false for nil vs non-empty")
	}
}

func TestDiffFilesEqual_BothNil(t *testing.T) {
	var a, b map[string]bool

	if !diffFilesEqual(a, b) {
		t.Error("expected true for both nil (len 0 == len 0)")
	}
}
