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

// TestSplitPaths tests path splitting functionality
func TestSplitPaths(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "single path",
			input:    "file.txt",
			expected: []string{"file.txt"},
		},
		{
			name:     "multiple paths space separated",
			input:    "file1.txt file2.txt file3.txt",
			expected: []string{"file1.txt", "file2.txt", "file3.txt"},
		},
		{
			name:     "multiple paths comma separated",
			input:    "file1.txt,file2.txt,file3.txt",
			expected: []string{"file1.txt", "file2.txt", "file3.txt"},
		},
		{
			name:     "mixed separators",
			input:    "file1.txt, file2.txt file3.txt",
			expected: []string{"file1.txt", "file2.txt", "file3.txt"},
		},
		{
			name:     "empty path",
			input:    "",
			expected: []string{"."},
		},
		{
			name:     "only spaces",
			input:    "   ",
			expected: []string{"."},
		},
		{
			name:     "path with directory",
			input:    "internal/tools/file.go cmd/main.go",
			expected: []string{"internal/tools/file.go", "cmd/main.go"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitPaths(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("splitPaths(%q) = %v, want %v", tt.input, result, tt.expected)
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("splitPaths(%q)[%d] = %q, want %q", tt.input, i, v, tt.expected[i])
				}
			}
		})
	}
}

// TestExecuteGitAdd_SingleVsBatch tests that single file uses single UI
func TestExecuteGitAdd_SingleVsBatch(t *testing.T) {
	// Single file should not trigger batch mode
	paths := splitPaths("single.txt")
	if len(paths) != 1 {
		t.Errorf("Single path should result in 1 element, got %d", len(paths))
	}

	// Multiple files should trigger batch mode
	paths = splitPaths("file1.txt file2.txt")
	if len(paths) != 2 {
		t.Errorf("Multiple paths should result in 2 elements, got %d", len(paths))
	}
}
