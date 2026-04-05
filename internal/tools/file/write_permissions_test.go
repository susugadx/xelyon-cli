package file

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/testutil"
)

func TestWriteFile_PreservesPermissions(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "executable.sh")

	testutil.CreateTempFile(t, tmpDir, "executable.sh", "#!/bin/bash\necho hello")
	if err := os.Chmod(testFile, 0755); err != nil {
		t.Fatalf("failed to set permissions: %v", err)
	}

	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("failed to stat file: %v", err)
	}
	if info.Mode().Perm() != 0755 {
		t.Fatalf("initial permissions should be 0755, got %o", info.Mode().Perm())
	}

	result, err := executeWriteFileForTest(testFile, "#!/bin/bash\necho world")
	if err != nil {
		t.Fatalf("executeWriteFileForTest failed: %v", err)
	}
	if !strings.Contains(result, "Successfully wrote") {
		t.Fatalf("expected success message, got: %s", result)
	}

	info, err = os.Stat(testFile)
	if err != nil {
		t.Fatalf("failed to stat file after write: %v", err)
	}
	if info.Mode().Perm() != 0755 {
		t.Errorf("permissions should be preserved as 0755, got %o", info.Mode().Perm())
	}

	testutil.AssertFileContent(t, testFile, "#!/bin/bash\necho world")
}
