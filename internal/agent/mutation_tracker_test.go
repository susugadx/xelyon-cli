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

func TestMutationTracker_RecordToolResult_UpdatesTurnMutationState(t *testing.T) {
	a := &Agent{
		agentWorkspaceState: agentWorkspaceState{
			changeStack: []tools.FileChange{},
		},
	}
	tracker := a.mutationTracker()
	state := newTurnMutationState()

	change := &tools.FileChange{
		FilePath: "/src/main.go",
		Tool:     "write_file",
		Details: []tools.FileChangeDetail{
			{FilePath: "/src/main.go", Action: "modified"},
			{FilePath: "/src/util.go", Action: "modified"},
		},
	}

	tracker.RecordToolResult(&tools.ToolCall{Tool: "write_file"}, "ok", change, &state)

	if !state.hasMutations() {
		t.Fatal("expected turn-local mutation state to be updated by file change event")
	}
	snapshot := state.snapshot()
	if snapshot.mutationCount != 1 {
		t.Fatalf("mutationCount = %d, want 1", snapshot.mutationCount)
	}
	if len(snapshot.files) != 2 {
		t.Fatalf("len(files) = %d, want 2", len(snapshot.files))
	}
	if snapshot.progressFingerprint == "" {
		t.Fatal("expected non-empty progress fingerprint")
	}
	if len(a.changeStack) != 1 {
		t.Fatalf("changeStack len = %d, want 1", len(a.changeStack))
	}
}

func TestTrackDeferredDiagnostics_StrReplacePrefersFileChangePath(t *testing.T) {
	t.Run("prefers detail path", func(t *testing.T) {
		a := &Agent{}
		tracker := a.mutationTracker()
		change := &tools.FileChange{
			FilePath: "display/path.go",
			Details: []tools.FileChangeDetail{
				{FilePath: "/resolved/detail.go"},
			},
		}

		tracker.trackDeferredDiagnostics(&tools.ToolCall{
			Tool: "str_replace",
			Args: map[string]string{"path": "args/fallback.go"},
		}, "Successfully replaced", change)

		if len(a.pendingLSPFiles) != 1 || a.pendingLSPFiles[0] != "/resolved/detail.go" {
			t.Fatalf("pendingLSPFiles = %v, want [/resolved/detail.go]", a.pendingLSPFiles)
		}
	})

	t.Run("falls back to change file path", func(t *testing.T) {
		a := &Agent{}
		tracker := a.mutationTracker()
		change := &tools.FileChange{FilePath: "/resolved/from-change.go"}

		tracker.trackDeferredDiagnostics(&tools.ToolCall{
			Tool: "str_replace",
			Args: map[string]string{"path": "args/fallback.go"},
		}, "Successfully replaced", change)

		if len(a.pendingLSPFiles) != 1 || a.pendingLSPFiles[0] != "/resolved/from-change.go" {
			t.Fatalf("pendingLSPFiles = %v, want [/resolved/from-change.go]", a.pendingLSPFiles)
		}
	})

	t.Run("falls back to tool args when change missing", func(t *testing.T) {
		a := &Agent{}
		tracker := a.mutationTracker()

		tracker.trackDeferredDiagnostics(&tools.ToolCall{
			Tool: "str_replace",
			Args: map[string]string{"path": "args/fallback.go"},
		}, "Successfully replaced", nil)

		if len(a.pendingLSPFiles) != 1 || a.pendingLSPFiles[0] != "args/fallback.go" {
			t.Fatalf("pendingLSPFiles = %v, want [args/fallback.go]", a.pendingLSPFiles)
		}
	})
}
