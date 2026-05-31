package turnsupport

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestMutationState_RecordFileChange_DeduplicatesFiles(t *testing.T) {
	state := NewMutationState()
	state.RecordFileChange(tools.FileChange{
		Tool: "apply_patch",
		Details: []tools.FileChangeDetail{
			{FilePath: "/src/a.go", Action: "modified"},
			{FilePath: "/src/a.go", Action: "modified"},
			{FilePath: "/src/b.go", Action: "created"},
		},
	})

	snapshot := state.Snapshot()
	if snapshot.MutationCount != 1 {
		t.Fatalf("MutationCount = %d, want 1", snapshot.MutationCount)
	}
	if len(snapshot.Files) != 2 {
		t.Fatalf("len(Files) = %d, want 2", len(snapshot.Files))
	}
}

func TestMutationState_ProgressFingerprintChangesOnAdditionalMutation(t *testing.T) {
	state := NewMutationState()
	state.RecordFileChange(tools.FileChange{FilePath: "/src/main.go", Tool: "write_file"})
	first := state.Snapshot().ProgressFingerprint
	if first == "" {
		t.Fatal("expected non-empty first fingerprint")
	}

	state.RecordFileChange(tools.FileChange{FilePath: "/src/main.go", Tool: "write_file"})
	second := state.Snapshot().ProgressFingerprint
	if second == "" {
		t.Fatal("expected non-empty second fingerprint")
	}
	if first == second {
		t.Fatal("expected fingerprint to change after additional mutation event")
	}
}

func TestSnapshotFileChanges_DeduplicatesFilesAndFingerprintsProgress(t *testing.T) {
	changes := []tools.FileChange{
		{
			Tool: "apply_patch",
			Details: []tools.FileChangeDetail{
				{FilePath: "/src/a.go", Action: "modified"},
				{FilePath: "/src/b.go", Action: "created"},
				{FilePath: "/src/a.go", Action: "modified"},
			},
		},
		{Tool: "write_file", FilePath: "/src/b.go"},
	}

	snapshot := SnapshotFileChanges(changes)
	if got, want := snapshot.Files, []string{"/src/a.go", "/src/b.go"}; len(got) != len(want) {
		t.Fatalf("Files = %v, want %v", got, want)
	} else {
		for i := range want {
			if got[i] != want[i] {
				t.Fatalf("Files = %v, want %v", got, want)
			}
		}
	}
	if snapshot.ProgressFingerprint == "" {
		t.Fatal("expected non-empty progress fingerprint")
	}

	advanced := append(append([]tools.FileChange(nil), changes...), tools.FileChange{
		Tool:     "write_file",
		FilePath: "/src/b.go",
	})
	if SnapshotFileChanges(advanced).ProgressFingerprint == snapshot.ProgressFingerprint {
		t.Fatal("expected fingerprint to change when a new FileChange is added")
	}
}
