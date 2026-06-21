package readtool

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/testutil"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

func TestExecuteReadFiles_DefaultBudgetPreservesFullContent(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()
	tmpDir := t.TempDir()

	paths := make([]string, 6)
	for i := range paths {
		name := fmt.Sprintf("f%d.txt", i)
		testutil.CreateTempFile(t, tmpDir, name, generateLines(200))
		paths[i] = filepath.Join(tmpDir, name)
	}

	output := ExecuteReadFiles(paths)

	if strings.Contains(output, "lines total") {
		t.Error("Default budget should not switch to outline mode for 200-line files")
	}
	if !strings.Contains(output, "200: line200") {
		t.Error("Expected full content with line 200")
	}
}

func TestExecuteReadFiles_SmallFilesFullContent(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()
	tmpDir := t.TempDir()

	paths := make([]string, 4)
	for i := range paths {
		name := fmt.Sprintf("f%d.txt", i)
		testutil.CreateTempFile(t, tmpDir, name, generateLines(50))
		paths[i] = filepath.Join(tmpDir, name)
	}

	output := ExecuteReadFiles(paths)

	if !strings.Contains(output, "50: line50") {
		t.Error("Expected full content with line 50")
	}
	if strings.Contains(output, "lines total") {
		t.Error("Small files should NOT have outline footer")
	}
}

func TestExecuteReadFiles_BudgetExplicitRangePriority(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()
	tmpDir := t.TempDir()

	paths := make([]string, 6)
	for i := 0; i < 5; i++ {
		name := fmt.Sprintf("f%d.txt", i)
		testutil.CreateTempFile(t, tmpDir, name, generateLines(200))
		paths[i] = filepath.Join(tmpDir, name)
	}
	testutil.CreateTempFile(t, tmpDir, "f5.txt", generateLines(200))
	paths[5] = filepath.Join(tmpDir, "f5.txt") + ":1-120"

	output := ExecuteReadFiles(paths)

	if !strings.Contains(output, "120: line120") {
		t.Error("Expected explicit range file to include line 120")
	}
	if strings.Contains(output, "lines total") {
		t.Error("Default budget should keep full content even when mixing explicit ranges")
	}
}

func TestExecuteReadFilesWithBudget_FullBudgetPreservesContent(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()
	tmpDir := t.TempDir()

	paths := make([]string, 3)
	for i := range paths {
		name := fmt.Sprintf("f%d.go", i)
		testutil.CreateTempFile(t, tmpDir, name, generateLines(180))
		paths[i] = filepath.Join(tmpDir, name)
	}

	normalOutput := ExecuteReadFiles(paths)
	if strings.Contains(normalOutput, "lines total") {
		t.Fatal("Expected normal 3-file read to keep full content for 180-line files")
	}

	fullOutput := ExecuteReadFilesWithBudget(common.DefaultOutput(), paths, DefaultFullLines)
	if strings.Contains(fullOutput, "lines total") {
		t.Error("ExecuteReadFilesWithBudget with DefaultFullLines should NOT use outline for 180-line files")
	}
	if !strings.Contains(fullOutput, "180: line180") {
		t.Error("Expected full content with line 180")
	}
}
