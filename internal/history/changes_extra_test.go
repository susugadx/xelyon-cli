package history

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/susugadx/xelyon-cli/internal/testutil"
)

func TestAppendChange_UsesDetailPaths(t *testing.T) {
	testutil.SetupTempHome(t)

	cs, err := NewChangeStorage()
	if err != nil {
		t.Fatalf("NewChangeStorage() error = %v", err)
	}

	change := ChangeRecordInput{
		FilePath:    "/tmp/fallback.txt",
		Timestamp:   time.Now(),
		Tool:        "apply_patch",
		Description: "updated files",
		Details: []ChangeDetail{
			{FilePath: "/tmp/a.txt"},
			{FilePath: "/tmp/b.txt"},
		},
	}
	if err := cs.AppendChange("session-details", change); err != nil {
		t.Fatalf("AppendChange() error = %v", err)
	}

	changes, err := cs.LoadSessionChanges("session-details")
	if err != nil {
		t.Fatalf("LoadSessionChanges() error = %v", err)
	}
	if len(changes) != 2 {
		t.Fatalf("len(changes) = %d, want 2", len(changes))
	}
	if changes[0].FilePath != "/tmp/a.txt" || changes[1].FilePath != "/tmp/b.txt" {
		t.Fatalf("changes paths = [%q, %q], want detail paths", changes[0].FilePath, changes[1].FilePath)
	}
}

func TestAppendChange_NoPathsLeavesEmptyFile(t *testing.T) {
	testutil.SetupTempHome(t)

	cs, err := NewChangeStorage()
	if err != nil {
		t.Fatalf("NewChangeStorage() error = %v", err)
	}

	if err := cs.AppendChange("session-empty", ChangeRecordInput{
		Timestamp:   time.Now(),
		Tool:        "apply_patch",
		Description: "no path change",
	}); err != nil {
		t.Fatalf("AppendChange() error = %v", err)
	}

	filename := filepath.Join(cs.changesPath, changeFileName("session-empty"))
	info, err := os.Stat(filename)
	if err != nil {
		t.Fatalf("changes file should exist as empty file, err = %v", err)
	}
	if info.Size() != 0 {
		t.Fatalf("changes file size = %d, want 0", info.Size())
	}
}
