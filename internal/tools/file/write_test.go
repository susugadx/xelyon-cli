package file

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/testutil"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

func TestExecuteWriteFile_NewFile(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "new.txt")
	testContent := "new file content"

	output, err := executeWriteFileForTest(testFile, testContent)

	if err != nil {
		t.Fatalf("ExecuteWriteFile failed: %v", err)
	}
	if !strings.Contains(output, "Successfully wrote") {
		t.Errorf("Expected success message, got: %s", output)
	}
	testutil.AssertFileExists(t, testFile)
	testutil.AssertFileContent(t, testFile, testContent)
}

func TestExecuteWriteFile_Overwrite(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "existing.txt")
	testutil.CreateTempFile(t, tmpDir, "existing.txt", "old content")

	output, err := executeWriteFileForTest(testFile, "new content")

	if err != nil {
		t.Fatalf("ExecuteWriteFile failed: %v", err)
	}
	if !strings.Contains(output, "Successfully wrote") {
		t.Errorf("Expected success message, got: %s", output)
	}
}

func TestExecuteWriteFile_UserCancelled(t *testing.T) {
	setupTestMocks(t)
	setupTestConfirm(t, false)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "cancelled.txt")

	output, err := executeWriteFileForTest(testFile, "content")

	if err != nil {
		t.Fatalf("ExecuteWriteFile should not error on cancel: %v", err)
	}
	if !strings.Contains(output, "Cancelled by user") {
		t.Errorf("Expected cancellation message, got: %s", output)
	}
	testutil.AssertFileNotExists(t, testFile)
}

func TestExecuteWriteFile_EmptyPath(t *testing.T) {
	output, err := executeWriteFileForTest("", "content")
	if err != nil {
		t.Fatalf("ExecuteWriteFile should not return error: %v", err)
	}
	if !strings.Contains(output, "Error: path is empty") {
		t.Errorf("Expected 'path is empty' error, got: %s", output)
	}
}

func TestExecuteWriteFile_PathTraversal(t *testing.T) {
	os.Setenv("XELYON_INTERACTIVE_CONFIRM", "0")
	t.Cleanup(func() { os.Unsetenv("XELYON_INTERACTIVE_CONFIRM") })
	setupTestConfirm(t, true)

	output, err := executeWriteFileForTest("../../../etc/passwd", "content")
	if err != nil {
		t.Fatalf("ExecuteWriteFile should not return error: %v", err)
	}
	if !strings.Contains(output, "Error:") {
		t.Errorf("Expected security error for path traversal, got: %s", output)
	}
}

func TestExecuteWriteFile_CreateDirectory(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "subdir", "nested", "file.txt")

	output, err := executeWriteFileForTest(testFile, "content")

	if err != nil {
		t.Fatalf("ExecuteWriteFile failed: %v", err)
	}
	if !strings.Contains(output, "Successfully wrote") {
		t.Errorf("Expected success message, got: %s", output)
	}
	testutil.AssertFileExists(t, testFile)
}

func TestExecuteWriteFile_OverwriteExistingFile(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "readable.txt")
	testutil.CreateTempFile(t, tmpDir, "readable.txt", "old content")

	output, err := executeWriteFileForTest(testFile, "new content")

	if err != nil {
		t.Fatalf("ExecuteWriteFile failed: %v", err)
	}
	if !strings.Contains(output, "Successfully wrote") {
		t.Errorf("Expected success message, got: %s", output)
	}
	testutil.AssertFileContent(t, testFile, "new content")
}

func TestExecuteWriteFile_AtomicWriteNoTempFileLeft(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "atomic.txt")
	testutil.CreateTempFile(t, tmpDir, "atomic.txt", "before")

	output, err := executeWriteFileForTest(testFile, "after")
	if err != nil {
		t.Fatalf("ExecuteWriteFile failed: %v", err)
	}
	if !strings.Contains(output, "Successfully wrote") {
		t.Errorf("Expected success message, got: %s", output)
	}
	testutil.AssertFileContent(t, testFile, "after")

	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("failed to read tmp directory: %v", err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".xelyon-write-") && strings.HasSuffix(e.Name(), ".tmp") {
			t.Fatalf("temporary file should not remain after successful write: %s", e.Name())
		}
	}
}

func TestExecuteWriteFile_SuccessMessageIncludesLineCount(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	file := filepath.Join(tmpDir, "test.txt")

	result, err := executeWriteFileForTest(file, "line1\nline2\nline3")
	if err != nil {
		t.Fatalf("ExecuteWriteFile failed: %v", err)
	}
	if !strings.Contains(result, "3 lines") {
		t.Fatalf("expected line count in result, got: %s", result)
	}
}

func TestExecuteWriteFile_CreatePreviewUsesAddOnlyPatchStyle(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "preview.txt")
	var stdout bytes.Buffer

	_, err := ExecuteWriteFileWithPromptIOAndOptionsAndLSPClient(
		testPromptIO(&stdout, &stdout),
		testConfirmOptions(),
		nil,
		testFile,
		"line1\nline2",
	)
	if err != nil {
		t.Fatalf("ExecuteWriteFile failed: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Creating") {
		t.Fatalf("expected patch-style create preview header, got: %s", output)
	}
	if !strings.Contains(output, "   1 + line1") || !strings.Contains(output, "   2 + line2") {
		t.Fatalf("expected add-only preview lines, got: %s", output)
	}
	if strings.Contains(output, "Preview:") {
		t.Fatalf("did not expect generic preview output, got: %s", output)
	}
}

func TestExecuteWriteFile_CreatePreviewUnderLimitShowsFullBody(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "full.txt")
	var stdout bytes.Buffer

	_, err := ExecuteWriteFileWithPromptIOAndOptionsAndLSPClient(
		testPromptIO(&stdout, &stdout),
		testConfirmOptions(),
		nil,
		testFile,
		"line1\nline2\nline3",
	)
	if err != nil {
		t.Fatalf("ExecuteWriteFile failed: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "   3 + line3") {
		t.Fatalf("expected full preview body under the cap, got: %s", output)
	}
	if strings.Contains(output, "preview truncated:") {
		t.Fatalf("did not expect truncation notice under the cap, got: %s", output)
	}
	if !strings.Contains(output, "(+3)") {
		t.Fatalf("expected metadata to use the real total line count, got: %s", output)
	}
}

func TestExecuteWriteFile_CreatePreviewLineCapKeepsRealAddedCount(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "large.txt")
	var stdout bytes.Buffer
	var builder strings.Builder
	const maxPreviewLines = 7
	for i := 1; i <= maxPreviewLines+5; i++ {
		if i > 1 {
			builder.WriteByte('\n')
		}
		_, _ = fmt.Fprintf(&builder, "line%d", i)
	}
	cfg := config.DefaultConfig()
	cfg.Diff.MaxTotalLines = maxPreviewLines

	_, err := ExecuteWriteFileWithPromptIOAndOptionsAndLSPClient(
		testPromptIO(&stdout, &stdout),
		common.ConfirmOptions{Config: cfg},
		nil,
		testFile,
		builder.String(),
	)
	if err != nil {
		t.Fatalf("ExecuteWriteFile failed: %v", err)
	}

	output := stdout.String()
	if strings.Contains(output, fmt.Sprintf("%4d + line%d", maxPreviewLines+1, maxPreviewLines+1)) {
		t.Fatalf("did not expect preview lines beyond the soft cap, got: %s", output)
	}
	if !strings.Contains(output, fmt.Sprintf("preview truncated: showing first %d of %d lines", maxPreviewLines, maxPreviewLines+5)) {
		t.Fatalf("expected explicit line-cap truncation notice, got: %s", output)
	}
	if !strings.Contains(output, fmt.Sprintf("(+%d)", maxPreviewLines+5)) {
		t.Fatalf("expected Added count to use the real total line count, got: %s", output)
	}
}

func TestExecuteWriteFile_CreatePreviewByteCapSignalsTruncation(t *testing.T) {
	setupTestMocks(t)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "bytes.txt")
	var stdout bytes.Buffer
	largeLine := strings.Repeat("a", maxFullBodyPreviewBytes/2)
	content := largeLine + "\n" + largeLine
	cfg := config.DefaultConfig()
	cfg.Diff.MaxTotalLines = 0

	_, err := ExecuteWriteFileWithPromptIOAndOptionsAndLSPClient(
		testPromptIO(&stdout, &stdout),
		common.ConfirmOptions{Config: cfg},
		nil,
		testFile,
		content,
	)
	if err != nil {
		t.Fatalf("ExecuteWriteFile failed: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "preview truncated at 64KB") {
		t.Fatalf("expected explicit byte-cap truncation notice, got: %s", output)
	}
	if !strings.Contains(output, "preview truncated: showing first 1 of 2 lines") {
		t.Fatalf("expected line summary for byte-cap truncation, got: %s", output)
	}
	if !strings.Contains(output, "(+2)") {
		t.Fatalf("expected Added count to keep the real total line count, got: %s", output)
	}
}
