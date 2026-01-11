package tools

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func setupGitRepo(t *testing.T) string {
	tmpDir := t.TempDir()

	// git init
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to initialize git repo: %v", err)
	}

	// git config
	exec.Command("git", "config", "user.name", "Test User").Run()
	exec.Command("git", "config", "user.email", "test@example.com").Run()

	// Change to tmpDir
	oldDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	t.Cleanup(func() { os.Chdir(oldDir) })

	return tmpDir
}

func TestExecuteGitAdd_Success(t *testing.T) {
	tmpDir := setupGitRepo(t)

	// Create a test file
	testFile := filepath.Join(tmpDir, "test.txt")
	if err := os.WriteFile(testFile, []byte("test content"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	output := executeGitAdd("test.txt")

	if strings.Contains(output, "Error:") {
		t.Errorf("executeGitAdd() should not error, got %v", output)
	}
}

func TestExecuteGitAdd_EmptyPath(t *testing.T) {
	// confirmをモック化（拒否）
	oldConfirm := confirm
	confirm = func(message string) bool { return false }
	t.Cleanup(func() { confirm = oldConfirm })

	setupGitRepo(t)

	output := executeGitAdd("")

	// Empty path will be cancelled by user or git will error
	if !strings.Contains(output, "Cancelled") && !strings.Contains(output, "Error:") {
		t.Errorf("executeGitAdd() output = %v, should be cancelled or error", output)
	}
}

func TestExecuteGitAdd_All(t *testing.T) {
	tmpDir := setupGitRepo(t)

	// Create multiple test files
	for i := 1; i <= 3; i++ {
		testFile := filepath.Join(tmpDir, "file"+string(rune('0'+i))+".txt")
		os.WriteFile(testFile, []byte("content"), 0644)
	}

	output := executeGitAdd(".")

	if strings.Contains(output, "Error:") {
		t.Errorf("executeGitAdd() should not error for '.', got %v", output)
	}
}
