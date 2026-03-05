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
	if !strings.Contains(output, "file1.txt") {
		t.Error("ExecuteListDir() output should contain 'file1.txt'")
	}
	if !strings.Contains(output, "file2.txt") {
		t.Error("ExecuteListDir() output should contain 'file2.txt'")
	}
	if !strings.Contains(output, "subdir") {
		t.Error("ExecuteListDir() output should contain 'subdir'")
	}
}

func TestExecuteListDir_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	chdirForListDirTest(t, tmpDir)
	output := ExecuteListDir(tmpDir, 1)

	if !strings.Contains(output, tmpDir) {
		t.Errorf("ExecuteListDir() output should contain directory path, got %v", output)
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

	origCfg := config.GetGlobalConfig()
	cfgCopy := *origCfg
	cfgCopy.ListDir.AdditionalIgnoreDirs = []string{"coverage"}
	config.SetGlobalConfig(&cfgCopy)
	t.Cleanup(func() { config.SetGlobalConfig(origCfg) })

	output := ExecuteListDir(tmpDir, 1)
	if strings.Contains(output, "coverage") {
		t.Errorf("coverage should be ignored by custom list_dir config, got: %s", output)
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
	if !strings.Contains(output, "child.txt") {
		t.Errorf("depth=2 should include child entries, got: %s", output)
	}
	if strings.Contains(output, "grandchild.txt") {
		t.Errorf("depth=2 should not include depth=3 entries, got: %s", output)
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
	if !strings.Contains(output, "showing first 200") {
		t.Errorf("expected truncation message, got: %s", output)
	}
}

func TestExecuteListDir_TreeConnectorWithIgnored(t *testing.T) {
	tmpDir := t.TempDir()
	chdirForListDirTest(t, tmpDir)
	if err := os.WriteFile(filepath.Join(tmpDir, "aaa.txt"), []byte("x"), 0644); err != nil {
		t.Fatalf("failed to create aaa.txt: %v", err)
	}
	if err := os.Mkdir(filepath.Join(tmpDir, "node_modules"), 0755); err != nil {
		t.Fatalf("failed to create ignored node_modules: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "bbb.txt"), []byte("x"), 0644); err != nil {
		t.Fatalf("failed to create bbb.txt: %v", err)
	}
	if err := os.Mkdir(filepath.Join(tmpDir, "vendor"), 0755); err != nil {
		t.Fatalf("failed to create ignored vendor: %v", err)
	}

	output := ExecuteListDir(tmpDir, 1)
	if !strings.Contains(output, "└── 📄 bbb.txt") {
		t.Errorf("last visible entry should use └── connector, got:\n%s", output)
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
