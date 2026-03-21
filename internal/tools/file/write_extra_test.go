package file

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/testutil"
)

func TestWriteFile_DerivativeFileBlocked(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()

	// Create the original file that the derivative would reference
	testutil.CreateTempFile(t, tmpDir, "main.go", "package main")

	derivativeSuffixes := []string{
		"main.go_temp",
		"main.go.new",
		"main.go_backup",
		"main.go.tmp",
		"main.go_copy",
		"main.go.bak",
	}

	for _, name := range derivativeSuffixes {
		t.Run(name, func(t *testing.T) {
			derivPath := filepath.Join(tmpDir, name)
			result, err := executeWriteFileForTest(derivPath, "content")
			if err != nil {
				t.Fatalf("executeWriteFileForTest should not return error: %v", err)
			}
			if !strings.Contains(result, "derivative") && !strings.Contains(result, "copy") {
				t.Errorf("expected derivative/copy error for %s, got: %s", name, result)
			}
			// Original file should be referenced in the error message
			if !strings.Contains(result, "main.go") {
				t.Errorf("expected reference to original file main.go, got: %s", result)
			}
			// The derivative file should not have been created
			testutil.AssertFileNotExists(t, derivPath)
		})
	}

	// Also verify that writing to a non-derivative path still works
	t.Run("non_derivative_succeeds", func(t *testing.T) {
		normalFile := filepath.Join(tmpDir, "other.go")
		result, err := executeWriteFileForTest(normalFile, "package other")
		if err != nil {
			t.Fatalf("executeWriteFileForTest failed: %v", err)
		}
		if !strings.Contains(result, "Successfully wrote") {
			t.Errorf("expected success for non-derivative file, got: %s", result)
		}
		testutil.AssertFileExists(t, normalFile)
	})

	// Verify derivative detection only triggers when the base file exists
	t.Run("derivative_allowed_when_base_missing", func(t *testing.T) {
		derivPath := filepath.Join(tmpDir, "nonexistent.go_temp")
		result, err := executeWriteFileForTest(derivPath, "content")
		if err != nil {
			t.Fatalf("executeWriteFileForTest failed: %v", err)
		}
		// When base file doesn't exist, derivative detection should not block
		if strings.Contains(result, "derivative") {
			t.Errorf("should not block derivative when base file doesn't exist, got: %s", result)
		}
	})
}

func TestWriteFile_PreservesPermissions(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "executable.sh")

	// Create file with executable permissions
	testutil.CreateTempFile(t, tmpDir, "executable.sh", "#!/bin/bash\necho hello")
	if err := os.Chmod(testFile, 0755); err != nil {
		t.Fatalf("failed to set permissions: %v", err)
	}

	// Verify initial permissions
	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("failed to stat file: %v", err)
	}
	if info.Mode().Perm() != 0755 {
		t.Fatalf("initial permissions should be 0755, got %o", info.Mode().Perm())
	}

	// Overwrite the file
	result, err := executeWriteFileForTest(testFile, "#!/bin/bash\necho world")
	if err != nil {
		t.Fatalf("executeWriteFileForTest failed: %v", err)
	}
	if !strings.Contains(result, "Successfully wrote") {
		t.Fatalf("expected success message, got: %s", result)
	}

	// Verify permissions are preserved
	info, err = os.Stat(testFile)
	if err != nil {
		t.Fatalf("failed to stat file after write: %v", err)
	}
	if info.Mode().Perm() != 0755 {
		t.Errorf("permissions should be preserved as 0755, got %o", info.Mode().Perm())
	}

	// Verify content was updated
	testutil.AssertFileContent(t, testFile, "#!/bin/bash\necho world")
}

func TestWriteFile_AtomicNoTempLeftover(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "atomic_test.txt")
	testutil.CreateTempFile(t, tmpDir, "atomic_test.txt", "original content")

	result, err := executeWriteFileForTest(testFile, "updated content")
	if err != nil {
		t.Fatalf("executeWriteFileForTest failed: %v", err)
	}
	if !strings.Contains(result, "Successfully wrote") {
		t.Fatalf("expected success message, got: %s", result)
	}

	// Verify target file content
	testutil.AssertFileContent(t, testFile, "updated content")

	// Verify no temp files remain in directory
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("failed to read directory: %v", err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".xelyon-write-") && strings.HasSuffix(name, ".tmp") {
			t.Errorf("temp file should not remain after successful write: %s", name)
		}
	}

	// There should be exactly one file in the directory
	if len(entries) != 1 {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Errorf("expected exactly 1 file in directory, got %d: %v", len(entries), names)
	}
}

func TestWriteFile_AtomicNoTempLeftover_NewFile(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "brand_new.txt")

	result, err := executeWriteFileForTest(testFile, "new content")
	if err != nil {
		t.Fatalf("executeWriteFileForTest failed: %v", err)
	}
	if !strings.Contains(result, "Successfully wrote") {
		t.Fatalf("expected success message, got: %s", result)
	}

	// Verify no temp files remain for new file creation either
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("failed to read directory: %v", err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".xelyon-write-") && strings.HasSuffix(name, ".tmp") {
			t.Errorf("temp file should not remain after creating new file: %s", name)
		}
	}
}
