package file

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/testutil"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

func TestExecuteReadFilesWithRuntime_NilConfigCache(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()
	tmpDir := t.TempDir()

	testutil.CreateTempFile(t, tmpDir, "a.go", "package main\nfunc main() {}")
	paths := []string{filepath.Join(tmpDir, "a.go")}

	output := ExecuteReadFilesWithRuntime(common.DefaultOutput(), nil, nil, paths, DefaultFullLines)
	if !strings.Contains(output, "package main") {
		t.Errorf("Expected output to contain file content, got: %s", output)
	}
}

func TestExecuteReadFilesWithRuntime_ConsistentWithSingleRead(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()
	tmpDir := t.TempDir()

	paths := make([]string, 3)
	for i := range paths {
		name := fmt.Sprintf("f%d.go", i)
		testutil.CreateTempFile(t, tmpDir, name, generateLines(150))
		paths[i] = filepath.Join(tmpDir, name)
	}

	singleOutput := ExecuteReadFileWithRuntime(common.DefaultOutput(), nil, nil, paths[0], 0, 0)
	if strings.Contains(singleOutput, "lines total") {
		t.Fatal("single read should show full content for 150-line file")
	}

	batchOutput := ExecuteReadFilesWithRuntime(common.DefaultOutput(), nil, nil, paths, DefaultFullLines)
	if strings.Contains(batchOutput, "lines total") {
		t.Error("batch read with DefaultFullLines should NOT use outline for 150-line files")
	}
	if !strings.Contains(batchOutput, "150: line150") {
		t.Error("Expected full content with line 150")
	}
}
