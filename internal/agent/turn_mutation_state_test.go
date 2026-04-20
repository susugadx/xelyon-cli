package agent

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestTurnMutationState_RecordFileChange_DeduplicatesFiles(t *testing.T) {
	state := newTurnMutationState()
	state.recordFileChange(tools.FileChange{
		Tool: "apply_patch",
		Details: []tools.FileChangeDetail{
			{FilePath: "/src/a.go", Action: "modified"},
			{FilePath: "/src/a.go", Action: "modified"},
			{FilePath: "/src/b.go", Action: "created"},
		},
	})

	snapshot := state.snapshot()
	if snapshot.mutationCount != 1 {
		t.Fatalf("mutationCount = %d, want 1", snapshot.mutationCount)
	}
	if len(snapshot.files) != 2 {
		t.Fatalf("len(files) = %d, want 2", len(snapshot.files))
	}
}

func TestTurnMutationState_ProgressFingerprintChangesOnAdditionalMutation(t *testing.T) {
	state := newTurnMutationState()
	state.recordFileChange(tools.FileChange{FilePath: "/src/main.go", Tool: "write_file"})
	first := state.snapshot().progressFingerprint
	if first == "" {
		t.Fatal("expected non-empty first fingerprint")
	}

	state.recordFileChange(tools.FileChange{FilePath: "/src/main.go", Tool: "write_file"})
	second := state.snapshot().progressFingerprint
	if second == "" {
		t.Fatal("expected non-empty second fingerprint")
	}
	if first == second {
		t.Fatal("expected fingerprint to change after additional mutation event")
	}
}
