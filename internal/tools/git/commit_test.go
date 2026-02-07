package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

// setupTestMocks sets up test environment
func setupTestMocks(t *testing.T) {
	t.Helper()
	os.Setenv("XELYON_INTERACTIVE_CONFIRM", "0")
	t.Cleanup(func() {
		os.Unsetenv("XELYON_INTERACTIVE_CONFIRM")
	})

	// Default to auto-approve
	setupTestConfirm(t, true)
}

// setupTestConfirm sets up test confirmation behavior
func setupTestConfirm(t *testing.T, result bool) {
	t.Helper()
	originalConfirm := common.SimpleConfirm
	common.SimpleConfirm = func(msg string) bool {
		return result
	}
	t.Cleanup(func() {
		common.SimpleConfirm = originalConfirm
	})
}

// setupGitRepo creates a temporary git repository for testing
func setupGitRepo(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change to temp directory: %v", err)
	}

	t.Cleanup(func() {
		_ = os.Chdir(originalDir)
	})

	// Initialize git repo
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Configure git user for commits
	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to configure git email: %v", err)
	}

	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to configure git name: %v", err)
	}

	return tmpDir
}

func TestExecuteGitCommit_Success(t *testing.T) {
	setupTestMocks(t)

	tmpDir := setupGitRepo(t)

	// Create and stage a file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cmd := exec.Command("git", "add", "test.txt")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to stage file: %v", err)
	}

	output := ExecuteGitCommit("Test commit message", true)

	if strings.Contains(output, "Error:") {
		t.Errorf("ExecuteGitCommit() should not error, got %v", output)
	}
	if !strings.Contains(output, "committed") && !strings.Contains(output, "create") {
		t.Errorf("ExecuteGitCommit() output = %v, should contain commit info", output)
	}
}

func TestExecuteGitCommit_EmptyMessage(t *testing.T) {
	setupGitRepo(t)

	output := ExecuteGitCommit("", true)

	if !strings.Contains(output, "Error:") || !strings.Contains(output, "required") {
		t.Errorf("ExecuteGitCommit() output = %v, should contain error about required message", output)
	}
}

func TestExecuteGitCommit_NothingToCommit(t *testing.T) {
	setupGitRepo(t)

	output := ExecuteGitCommit("Test commit", true)

	// Nothing staged, should still execute but git will say nothing to commit
	// We check that it doesn't crash
	_ = output
}

func TestExecuteGitCommit_UserCancelled(t *testing.T) {
	setupTestMocks(t)
	setupTestConfirm(t, false)

	setupGitRepo(t)

	output := ExecuteGitCommit("Test commit", true)

	if !strings.Contains(output, "Cancelled") {
		t.Errorf("ExecuteGitCommit() should be cancelled, got %v", output)
	}
}

func TestExecuteGitCommit_DiagnosticsSkipped(t *testing.T) {
	setupTestMocks(t)

	tmpDir := setupGitRepo(t)

	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cmd := exec.Command("git", "add", "test.txt")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to stage file: %v", err)
	}

	// skipDiagnostics=true でコミットが成功することを確認
	output := ExecuteGitCommit("Test with diagnostics skipped", true)
	if strings.Contains(output, "BLOCKED") {
		t.Errorf("ExecuteGitCommit(skipDiagnostics=true) should not be blocked, got %v", output)
	}
}

func TestExecuteGitCommit_DiagnosticsRun_NoLSP(t *testing.T) {
	setupTestMocks(t)

	tmpDir := setupGitRepo(t)

	testFile := filepath.Join(tmpDir, "hello.go")
	if err := os.WriteFile(testFile, []byte("package main\n"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	cmd := exec.Command("git", "add", "hello.go")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to stage file: %v", err)
	}

	// LSP未起動時（LSPClient==nil）はブロックされずにコミットが進むことを確認
	output := ExecuteGitCommit("Test with diagnostics enabled but no LSP", false)
	if strings.Contains(output, "BLOCKED") {
		t.Errorf("ExecuteGitCommit() should not block when LSP is unavailable, got %v", output)
	}
}

func TestGetStagedFiles(t *testing.T) {
	setupGitRepo(t)

	// ファイルなし
	files, err := getStagedFiles()
	if err != nil {
		t.Fatalf("getStagedFiles() error: %v", err)
	}
	if len(files) != 0 {
		t.Errorf("getStagedFiles() = %v, want empty", files)
	}
}

func TestGetStagedFiles_WithFiles(t *testing.T) {
	tmpDir := setupGitRepo(t)

	// ファイルを作成してステージ
	for _, name := range []string{"a.go", "b.go"} {
		path := filepath.Join(tmpDir, name)
		if err := os.WriteFile(path, []byte("package main\n"), 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
		cmd := exec.Command("git", "add", name)
		cmd.Dir = tmpDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to stage file: %v", err)
		}
	}

	files, err := getStagedFiles()
	if err != nil {
		t.Fatalf("getStagedFiles() error: %v", err)
	}
	if len(files) != 2 {
		t.Errorf("getStagedFiles() = %d files, want 2", len(files))
	}
}
