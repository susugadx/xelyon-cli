package file

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/pathmatch"
)

func TestSummarizeListDir_BuildsStructuredSummary(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, "a", "nested"), 0755); err != nil {
		t.Fatalf("failed to create nested dir: %v", err)
	}
	if err := os.Mkdir(filepath.Join(tmpDir, "b"), 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "root.txt"), []byte("root"), 0644); err != nil {
		t.Fatalf("failed to write root file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "a", "child.txt"), []byte("child"), 0644); err != nil {
		t.Fatalf("failed to write child file: %v", err)
	}

	section := summarizeListDir(tmpDir, tmpDir, "", 2, pathmatch.NewMatcher(nil), &listDirBudget{remainingEntries: maxEntries}, true)

	if section.totalDirs != 2 || section.totalFiles != 1 {
		t.Fatalf("unexpected root totals: dirs=%d files=%d", section.totalDirs, section.totalFiles)
	}
	if len(section.dirs) != 2 || section.dirs[0] != "a/" || section.dirs[1] != "b/" {
		t.Fatalf("unexpected root dirs: %#v", section.dirs)
	}
	if len(section.files) != 1 || section.files[0].name != "root.txt" {
		t.Fatalf("unexpected root files: %#v", section.files)
	}
	if len(section.subtrees) != 2 {
		t.Fatalf("expected 2 subtrees, got %d", len(section.subtrees))
	}

	child := section.subtrees[0]
	if child.relPath != "a/" {
		t.Fatalf("unexpected child relPath: %s", child.relPath)
	}
	if child.totalDirs != 1 || child.totalFiles != 1 {
		t.Fatalf("unexpected child totals: dirs=%d files=%d", child.totalDirs, child.totalFiles)
	}
	if len(child.files) != 1 || child.files[0].name != "child.txt" {
		t.Fatalf("unexpected child files: %#v", child.files)
	}
}

func TestSummarizeListDir_TracksMoreCounts(t *testing.T) {
	tmpDir := t.TempDir()
	for i := 0; i < 10; i++ {
		if err := os.Mkdir(filepath.Join(tmpDir, fmt.Sprintf("dir-%02d", i)), 0755); err != nil {
			t.Fatalf("failed to create dir %d: %v", i, err)
		}
	}
	for i := 0; i < 10; i++ {
		name := filepath.Join(tmpDir, fmt.Sprintf("file-%02d.txt", i))
		if err := os.WriteFile(name, []byte("x"), 0644); err != nil {
			t.Fatalf("failed to create file %d: %v", i, err)
		}
	}

	section := summarizeListDir(tmpDir, tmpDir, "", 1, pathmatch.NewMatcher(nil), &listDirBudget{remainingEntries: maxEntries}, true)

	if section.moreDirs != 2 {
		t.Fatalf("expected 2 extra dirs, got %d", section.moreDirs)
	}
	if section.moreFiles != 2 {
		t.Fatalf("expected 2 extra files, got %d", section.moreFiles)
	}
}
