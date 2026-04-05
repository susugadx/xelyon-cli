package file

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/pathmatch"
)

func TestSummarizeListDirSubtrees_RootLimitAndMoreCount(t *testing.T) {
	tmpDir := t.TempDir()
	for i := 0; i < maxRootSubtreesShown+2; i++ {
		dir := filepath.Join(tmpDir, fmt.Sprintf("dir-%02d", i))
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create dir %d: %v", i, err)
		}
		if err := os.WriteFile(filepath.Join(dir, "child.txt"), []byte("x"), 0644); err != nil {
			t.Fatalf("failed to create child file %d: %v", i, err)
		}
	}

	dirs, _, err := readVisibleListDirEntries(tmpDir, tmpDir, pathmatch.NewMatcher(nil))
	if err != nil {
		t.Fatalf("readVisibleListDirEntries failed: %v", err)
	}

	subtrees, more := summarizeListDirSubtrees(tmpDir, tmpDir, "", 2, pathmatch.NewMatcher(nil), &listDirBudget{remainingEntries: maxEntries}, dirs)
	if len(subtrees) != maxRootSubtreesShown {
		t.Fatalf("expected %d subtrees, got %d", maxRootSubtreesShown, len(subtrees))
	}
	if more != 2 {
		t.Fatalf("expected 2 remaining subtrees, got %d", more)
	}
	if subtrees[0].relPath != "dir-00/" {
		t.Fatalf("unexpected first relPath: %s", subtrees[0].relPath)
	}
}

func TestSummarizeListDirSubtrees_StopsWhenBudgetExhausted(t *testing.T) {
	tmpDir := t.TempDir()
	for i := 0; i < 2; i++ {
		dir := filepath.Join(tmpDir, fmt.Sprintf("dir-%02d", i))
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create dir %d: %v", i, err)
		}
	}

	dirs, _, err := readVisibleListDirEntries(tmpDir, tmpDir, pathmatch.NewMatcher(nil))
	if err != nil {
		t.Fatalf("readVisibleListDirEntries failed: %v", err)
	}

	subtrees, more := summarizeListDirSubtrees(tmpDir, tmpDir, "", 2, pathmatch.NewMatcher(nil), &listDirBudget{remainingEntries: 0}, dirs)
	if len(subtrees) != 0 {
		t.Fatalf("expected no expanded subtrees, got %d", len(subtrees))
	}
	if more != len(dirs) {
		t.Fatalf("expected %d remaining subtrees, got %d", len(dirs), more)
	}
}

func TestSummarizeListDirSubtrees_PreservesNestedRelPath(t *testing.T) {
	tmpDir := t.TempDir()
	parentDir := filepath.Join(tmpDir, "parent")
	childDir := filepath.Join(parentDir, "child")
	if err := os.MkdirAll(childDir, 0755); err != nil {
		t.Fatalf("failed to create nested dir: %v", err)
	}

	dirs, _, err := readVisibleListDirEntries(parentDir, tmpDir, pathmatch.NewMatcher(nil))
	if err != nil {
		t.Fatalf("readVisibleListDirEntries failed: %v", err)
	}

	subtrees, more := summarizeListDirSubtrees(parentDir, tmpDir, "parent/", 2, pathmatch.NewMatcher(nil), &listDirBudget{remainingEntries: maxEntries}, dirs)
	if more != 0 {
		t.Fatalf("expected no remaining subtrees, got %d", more)
	}
	if len(subtrees) != 1 {
		t.Fatalf("expected one subtree, got %d", len(subtrees))
	}
	if subtrees[0].relPath != "parent/child/" {
		t.Fatalf("unexpected nested relPath: %s", subtrees[0].relPath)
	}
}
