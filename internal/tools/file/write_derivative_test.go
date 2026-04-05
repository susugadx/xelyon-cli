package file

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/testutil"
)

func TestWriteFile_DerivativeFileBlocked(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
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
			if !strings.Contains(result, "main.go") {
				t.Errorf("expected reference to original file main.go, got: %s", result)
			}
			testutil.AssertFileNotExists(t, derivPath)
		})
	}

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

	t.Run("derivative_allowed_when_base_missing", func(t *testing.T) {
		derivPath := filepath.Join(tmpDir, "nonexistent.go_temp")
		result, err := executeWriteFileForTest(derivPath, "content")
		if err != nil {
			t.Fatalf("executeWriteFileForTest failed: %v", err)
		}
		if strings.Contains(result, "derivative") {
			t.Errorf("should not block derivative when base file doesn't exist, got: %s", result)
		}
	})
}
