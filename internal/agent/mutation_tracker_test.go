package agent

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestAddPendingLSPFile(t *testing.T) {
	a := &Agent{}

	a.addPendingLSPFile("")
	if len(a.pendingLSPFiles) != 0 {
		t.Errorf("expected 0 files after adding empty path, got %d", len(a.pendingLSPFiles))
	}

	a.addPendingLSPFile("/src/main.go")
	if len(a.pendingLSPFiles) != 1 || a.pendingLSPFiles[0] != "/src/main.go" {
		t.Errorf("expected [/src/main.go], got %v", a.pendingLSPFiles)
	}

	a.addPendingLSPFile("/src/main.go")
	if len(a.pendingLSPFiles) != 1 {
		t.Errorf("expected 1 file after duplicate add, got %d", len(a.pendingLSPFiles))
	}

	a.addPendingLSPFile("/src/util.go")
	if len(a.pendingLSPFiles) != 2 {
		t.Errorf("expected 2 files, got %d", len(a.pendingLSPFiles))
	}
}

func TestAddPendingLSPFilesFromChange(t *testing.T) {
	t.Run("nil change is safe", func(t *testing.T) {
		a := &Agent{}
		a.addPendingLSPFilesFromChange(nil)
		if len(a.pendingLSPFiles) != 0 {
			t.Errorf("expected 0 files, got %d", len(a.pendingLSPFiles))
		}
	})

	t.Run("empty details is safe", func(t *testing.T) {
		a := &Agent{}
		a.addPendingLSPFilesFromChange(&tools.FileChange{})
		if len(a.pendingLSPFiles) != 0 {
			t.Errorf("expected 0 files, got %d", len(a.pendingLSPFiles))
		}
	})

	t.Run("single file from apply_patch", func(t *testing.T) {
		a := &Agent{}
		a.addPendingLSPFilesFromChange(&tools.FileChange{
			Tool: "apply_patch",
			Details: []tools.FileChangeDetail{
				{FilePath: "/src/main.go", Action: "modified"},
			},
		})
		if len(a.pendingLSPFiles) != 1 || a.pendingLSPFiles[0] != "/src/main.go" {
			t.Errorf("expected [/src/main.go], got %v", a.pendingLSPFiles)
		}
	})

	t.Run("multiple files from apply_patch", func(t *testing.T) {
		a := &Agent{}
		a.addPendingLSPFilesFromChange(&tools.FileChange{
			Tool: "apply_patch",
			Details: []tools.FileChangeDetail{
				{FilePath: "/src/main.go", Action: "modified"},
				{FilePath: "/src/util.go", Action: "created"},
				{FilePath: "/src/old.go", Action: "deleted"},
			},
		})
		if len(a.pendingLSPFiles) != 3 {
			t.Errorf("expected 3 files, got %d", len(a.pendingLSPFiles))
		}
	})

	t.Run("dedup with existing pending files", func(t *testing.T) {
		a := &Agent{}
		a.addPendingLSPFile("/src/main.go")
		a.addPendingLSPFilesFromChange(&tools.FileChange{
			Tool: "apply_patch",
			Details: []tools.FileChangeDetail{
				{FilePath: "/src/main.go", Action: "modified"},
				{FilePath: "/src/new.go", Action: "created"},
			},
		})
		if len(a.pendingLSPFiles) != 2 {
			t.Errorf("expected 2 files (dedup), got %d: %v", len(a.pendingLSPFiles), a.pendingLSPFiles)
		}
	})
}
