package mutation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/testutil"
)

func TestWriteFile_AtomicNoTempLeftover(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

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

	testutil.AssertFileContent(t, testFile, "updated content")

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
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "brand_new.txt")

	result, err := executeWriteFileForTest(testFile, "new content")
	if err != nil {
		t.Fatalf("executeWriteFileForTest failed: %v", err)
	}
	if !strings.Contains(result, "Successfully wrote") {
		t.Fatalf("expected success message, got: %s", result)
	}

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
