package file

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
)

func chdirForListDirTest(t *testing.T, dir string) {
	t.Helper()
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWd) })
}

func TestExecuteListDir_Success(t *testing.T) {
	tmpDir := t.TempDir()
	chdirForListDirTest(t, tmpDir)

	if err := os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("content1"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "file2.txt"), []byte("content2"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if err := os.Mkdir(filepath.Join(tmpDir, "subdir"), 0755); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	output := ExecuteListDir(tmpDir, 1)

	if !strings.Contains(output, tmpDir) {
		t.Errorf("ExecuteListDir() output should contain directory path, got %v", output)
	}
	if !strings.Contains(output, "summary: depth=1, dirs=1, files=2") {
		t.Errorf("expected compact summary header, got %s", output)
	}
	if !strings.Contains(output, "dirs: subdir/") {
		t.Errorf("expected directory summary, got %s", output)
	}
	if !strings.Contains(output, "file1.txt") || !strings.Contains(output, "file2.txt") {
		t.Errorf("expected representative file names, got %s", output)
	}
}

func TestExecuteListDir_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	chdirForListDirTest(t, tmpDir)
	output := ExecuteListDir(tmpDir, 1)

	if !strings.Contains(output, tmpDir) {
		t.Errorf("ExecuteListDir() output should contain directory path, got %v", output)
	}
	if !strings.Contains(output, "summary: depth=1, dirs=0, files=0") {
		t.Errorf("expected empty summary, got %s", output)
	}
}

func TestExecuteListDir_DirectoryNotFound(t *testing.T) {
	tmpDir := t.TempDir()
	chdirForListDirTest(t, tmpDir)
	nonExistentDir := filepath.Join(tmpDir, "notexist")

	output := ExecuteListDir(nonExistentDir, 1)

	if !strings.Contains(output, "Error") {
		t.Errorf("ExecuteListDir() output = %v, should contain 'Error'", output)
	}
}

func TestExecuteListDir_FileSizes(t *testing.T) {
	tmpDir := t.TempDir()
	chdirForListDirTest(t, tmpDir)

	if err := os.WriteFile(filepath.Join(tmpDir, "small.txt"), []byte("small"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "large.txt"), []byte(strings.Repeat("x", 1000)), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	output := ExecuteListDir(tmpDir, 1)

	if !strings.Contains(output, "bytes") {
		t.Error("ExecuteListDir() output should contain file size in bytes")
	}
}

func TestExecuteListDir_RelativePath(t *testing.T) {
	output := ExecuteListDir(".", 1)

	absPath, err := filepath.Abs(".")
	if err != nil {
		t.Fatalf("Failed to get absolute path: %v", err)
	}

	if !strings.Contains(output, absPath) {
		t.Errorf("ExecuteListDir() output should contain absolute path, got %v", output)
	}
}

func TestExecuteListDir_IgnoreDefaultDirs(t *testing.T) {
	tmpDir := t.TempDir()
	chdirForListDirTest(t, tmpDir)
	if err := os.Mkdir(filepath.Join(tmpDir, "node_modules"), 0755); err != nil {
		t.Fatalf("Failed to create ignored directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "keep.txt"), []byte("ok"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	output := ExecuteListDir(tmpDir, 2)
	if strings.Contains(output, "node_modules") {
		t.Errorf("node_modules should be ignored, got: %s", output)
	}
	if !strings.Contains(output, "keep.txt") {
		t.Errorf("keep.txt should be listed, got: %s", output)
	}
}

func TestExecuteListDir_CustomIgnoreDirs(t *testing.T) {
	tmpDir := t.TempDir()
	chdirForListDirTest(t, tmpDir)

	if err := os.Mkdir(filepath.Join(tmpDir, "coverage"), 0755); err != nil {
		t.Fatalf("Failed to create coverage directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "keep.txt"), []byte("ok"), 0644); err != nil {
		t.Fatalf("Failed to create keep file: %v", err)
	}

	cfg := config.DefaultConfig()
	cfg.ListDir.AdditionalIgnoreDirs = []string{"coverage"}

	output := ExecuteListDirWithRuntime(cfg, nil, tmpDir, 1)
	if strings.Contains(output, "coverage") {
		t.Errorf("coverage should be ignored by custom list_dir config, got: %s", output)
	}
	if !strings.Contains(output, "keep.txt") {
		t.Errorf("keep.txt should be listed, got: %s", output)
	}
}

func TestExecuteListDir_ProjectIgnorePatterns(t *testing.T) {
	tmpDir := t.TempDir()
	chdirForListDirTest(t, tmpDir)

	if err := os.WriteFile(filepath.Join(tmpDir, "xelyon.yaml"), []byte("ignore:\n  patterns:\n    - coverage\n"), 0644); err != nil {
		t.Fatalf("Failed to create xelyon.yaml: %v", err)
	}
	if err := os.Mkdir(filepath.Join(tmpDir, "coverage"), 0755); err != nil {
		t.Fatalf("Failed to create coverage directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "keep.txt"), []byte("ok"), 0644); err != nil {
		t.Fatalf("Failed to create keep file: %v", err)
	}

	output := ExecuteListDirWithRuntime(config.DefaultConfig(), nil, tmpDir, 1)
	if strings.Contains(output, "coverage") {
		t.Errorf("coverage should be ignored by xelyon.yaml ignore.patterns, got: %s", output)
	}
	if !strings.Contains(output, "keep.txt") {
		t.Errorf("keep.txt should be listed, got: %s", output)
	}
}

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

func TestExecuteListDir_RejectsOutsideWorkspace(t *testing.T) {
	tmpDir := t.TempDir()
	chdirForListDirTest(t, tmpDir)

	output := ExecuteListDir("/etc", 1)
	if !strings.Contains(output, "Error") {
		t.Errorf("expected error for path outside workspace, got: %s", output)
	}
}
