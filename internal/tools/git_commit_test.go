package tools

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteGitCommit_Success(t *testing.T) {
	// confirmをモック化
	oldConfirm := confirm
	confirm = func(message string) bool { return true }
	t.Cleanup(func() { confirm = oldConfirm })

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
