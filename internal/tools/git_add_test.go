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

	// git config (must run in tmpDir for local config)
	configName := exec.Command("git", "config", "user.name", "Test User")
	configName.Dir = tmpDir
	_ = configName.Run()
	configEmail := exec.Command("git", "config", "user.email", "test@example.com")
	configEmail.Dir = tmpDir
	_ = configEmail.Run()

	// Change to tmpDir
	oldDir, _ := os.Getwd()
	_ = os.Chdir(tmpDir)
	t.Cleanup(func() { _ = os.Chdir(oldDir) })

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
	setupTestMocks(t)
	// confirmをモック化（拒否）
	setupTestConfirm(t, false)

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
		_ = os.WriteFile(testFile, []byte("content"), 0644)
	}

	output := executeGitAdd(".")

	if strings.Contains(output, "Error:") {
		t.Errorf("executeGitAdd() should not error for '.', got %v", output)
	}
}
