package file

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteListDir_DepthTwoShowsChildren(t *testing.T) {
	tmpDir := t.TempDir()
	chdirForListDirTest(t, tmpDir)
	if err := os.MkdirAll(filepath.Join(tmpDir, "a", "b"), 0755); err != nil {
		t.Fatalf("Failed to create nested directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "a", "child.txt"), []byte("x"), 0644); err != nil {
		t.Fatalf("Failed to create nested file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "a", "b", "grandchild.txt"), []byte("x"), 0644); err != nil {
		t.Fatalf("Failed to create deeper nested file: %v", err)
	}

	output := ExecuteListDir(tmpDir, 2)
	if !strings.Contains(output, "subtrees: 1 shown") {
		t.Errorf("depth=2 should include subtree summary, got: %s", output)
	}
	if !strings.Contains(output, "- a/ -> dirs=1, files=1") {
		t.Errorf("depth=2 should include child directory summary, got: %s", output)
	}
	if !strings.Contains(output, "child.txt") {
		t.Errorf("depth=2 should include child entries, got: %s", output)
	}
	if strings.Contains(output, "grandchild.txt") {
		t.Errorf("depth=2 should not include depth=3 entries, got: %s", output)
	}
}

func TestExecuteListDir_DepthThreeShowsNestedHints(t *testing.T) {
	tmpDir := t.TempDir()
	chdirForListDirTest(t, tmpDir)
	if err := os.MkdirAll(filepath.Join(tmpDir, "a", "b"), 0755); err != nil {
		t.Fatalf("Failed to create nested directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "a", "b", "grandchild.txt"), []byte("x"), 0644); err != nil {
		t.Fatalf("Failed to create deeper nested file: %v", err)
	}

	output := ExecuteListDir(tmpDir, 3)
	if !strings.Contains(output, "- a/b/ -> dirs=0, files=1") {
		t.Errorf("depth=3 should include nested subtree hint, got: %s", output)
	}
	if !strings.Contains(output, "grandchild.txt") {
		t.Errorf("depth=3 should include grandchild entry, got: %s", output)
	}
}

func TestExecuteListDir_TruncatesEntries(t *testing.T) {
	tmpDir := t.TempDir()
	chdirForListDirTest(t, tmpDir)
	for i := 0; i < 210; i++ {
		name := filepath.Join(tmpDir, fmt.Sprintf("f%03d.txt", i))
		if err := os.WriteFile(name, []byte("x"), 0644); err != nil {
			t.Fatalf("Failed to create file %d: %v", i, err)
		}
	}

	output := ExecuteListDir(tmpDir, 1)
	if !strings.Contains(output, "files: f000.txt (1 bytes)") {
		t.Errorf("expected representative files, got: %s", output)
	}
	if !strings.Contains(output, "(+202 more)") {
		t.Errorf("expected compact remainder count, got: %s", output)
	}
	if strings.Contains(output, "f209.txt") {
		t.Errorf("expected compact summary instead of full expansion, got: %s", output)
	}
}

func TestExecuteListDir_WideDepthTwoKeepsSubtreeHintsCompact(t *testing.T) {
	tmpDir := t.TempDir()
	chdirForListDirTest(t, tmpDir)
	for i := 0; i < 9; i++ {
		dir := filepath.Join(tmpDir, fmt.Sprintf("dir-%02d", i))
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("failed to create %s: %v", dir, err)
		}
		file := filepath.Join(dir, "child.txt")
		if err := os.WriteFile(file, []byte("x"), 0644); err != nil {
			t.Fatalf("failed to create %s: %v", file, err)
		}
	}

	output := ExecuteListDir(tmpDir, 2)
	if !strings.Contains(output, "dirs: dir-00/, dir-01/, dir-02/, dir-03/, dir-04/, dir-05/, dir-06/, dir-07/, (+1 more)") {
		t.Errorf("expected compact top-level directory list, got: %s", output)
	}
	if !strings.Contains(output, "subtrees: 6 shown (+3 more)") {
		t.Errorf("expected limited subtree expansion, got: %s", output)
	}
	if strings.Contains(output, "- dir-08/ ->") {
		t.Errorf("expected subtree details to stop after budgeted representative dirs, got: %s", output)
	}
}
