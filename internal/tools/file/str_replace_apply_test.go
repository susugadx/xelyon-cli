package file

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/testutil"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

func TestApplyStringReplaceMutation_Success(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testutil.CreateTempFile(t, tmpDir, "test.txt", "before")

	var out bytes.Buffer
	ctx := fileMutationContext{
		promptIO: testPromptIO(&out, &out),
		out:      common.NewOutput(&out, &out),
		path:     testFile,
		absPath:  testFile,
	}

	result, err := applyStringReplaceMutation(ctx, "after", "✅ Replaced in: test.txt", "Successfully replaced text in test.txt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.status != fileMutationStatusApplied {
		t.Fatalf("expected applied result, got: %+v", result)
	}
	if result.message != "Successfully replaced text in test.txt" {
		t.Fatalf("unexpected result message: %s", result.message)
	}
	if !strings.Contains(out.String(), "✅ Replaced in: test.txt") {
		t.Fatalf("expected success stdout message, got: %q", out.String())
	}

	updated, readErr := os.ReadFile(testFile)
	if readErr != nil {
		t.Fatalf("failed to read file: %v", readErr)
	}
	if string(updated) != "after" {
		t.Fatalf("unexpected file content: %q", string(updated))
	}
}

func TestApplyStringReplaceMutation_WriteError(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	var out bytes.Buffer
	ctx := fileMutationContext{
		promptIO: testPromptIO(&out, &out),
		out:      common.NewOutput(&out, &out),
		path:     tmpDir,
		absPath:  tmpDir, // directory path to force write failure
	}

	result, err := applyStringReplaceMutation(ctx, "after", "ignored", "ignored")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.status != fileMutationStatusError {
		t.Fatalf("expected error result, got: %+v", result)
	}
	if !strings.Contains(result.message, "Error writing file:") {
		t.Fatalf("unexpected error message: %s", result.message)
	}
}
