package readtool

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/locator"
	"github.com/susugadx/xelyon-cli/internal/testutil"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

func TestReadFileTool_Targets(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "test.go", "package main\n\nfunc main() {\n\tprintln(\"hello\")\n}\n")

	// Registryに登録
	reg := locator.NewRegistry()
	id := reg.Register(locator.Location{
		FilePath: tmpDir + "/test.go",
		Line:     3,
		EndLine:  5,
	})

	if id != "[L1]" {
		t.Fatalf("expected [L1], got %s", id)
	}

	tool := &ReadFileTool{}
	execCtx := tools.ExecutionContext{
		LocatorRegistry: reg,
	}

	result, _, err := tool.Run(execCtx, map[string]string{
		"targets": "[L1]",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "func main()") {
		t.Errorf("expected file content with func main(), got:\n%s", result)
	}
}

func TestReadFileTool_TargetsInvalidID(t *testing.T) {
	reg := locator.NewRegistry()

	tool := &ReadFileTool{}
	execCtx := tools.ExecutionContext{
		LocatorRegistry: reg,
	}

	result, _, err := tool.Run(execCtx, map[string]string{
		"targets": "[L99]",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "Error: no valid locator IDs found") {
		t.Errorf("expected error for invalid targets, got: %s", result)
	}
}

func TestReadFileTool_LocatorIDInOutput(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "locator_test.txt", "line1\nline2\nline3\n")

	reg := locator.NewRegistry()

	result := ExecuteReadFilesWithLocator(
		common.DefaultOutput(),
		nil, nil,
		[]string{tmpDir + "/locator_test.txt"},
		DefaultFullLines,
		reg,
	)

	if !strings.Contains(result, "[L1]") {
		t.Errorf("expected Locator ID in output, got:\n%s", result)
	}
}

func TestReadFileTool_LocatorIDNilRegistry(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "no_locator.txt", "line1\nline2\n")

	// nil registry → IDなし
	result := ExecuteReadFilesWithRuntime(
		common.DefaultOutput(),
		nil, nil,
		[]string{tmpDir + "/no_locator.txt"},
		DefaultFullLines,
	)

	if strings.Contains(result, "[L") {
		t.Errorf("expected no Locator ID with nil registry, got:\n%s", result)
	}
}
