package tools

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

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

	output := executeGitCommit("Test commit message")

	if strings.Contains(output, "Error:") {
		t.Errorf("executeGitCommit() should not error, got %v", output)
	}
	if !strings.Contains(output, "committed") && !strings.Contains(output, "create") {
		t.Errorf("executeGitCommit() output = %v, should contain commit info", output)
	}
}

func TestExecuteGitCommit_EmptyMessage(t *testing.T) {
	setupGitRepo(t)

	output := executeGitCommit("")

	if !strings.Contains(output, "Error:") || !strings.Contains(output, "required") {
		t.Errorf("executeGitCommit() output = %v, should contain error about required message", output)
	}
}

func TestExecuteGitCommit_NothingToCommit(t *testing.T) {
	setupGitRepo(t)

	output := executeGitCommit("Test commit")

	// Nothing staged, should still execute but git will say nothing to commit
	// We check that it doesn't crash
	_ = output
}
