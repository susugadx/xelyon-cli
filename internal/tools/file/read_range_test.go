package file

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/testutil"
)

func TestExecuteReadFile_LineRange(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "range.txt", "line1\nline2\nline3\nline4\nline5")

	output := ExecuteReadFile(filepath.Join(tmpDir, "range.txt"), 2, 4)

	if !strings.Contains(output, "line2") || !strings.Contains(output, "line4") {
		t.Errorf("Expected output to contain lines 2-4, got: %s", output)
	}
}

func TestExecuteReadFile_StartLineOnly(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()
	tmpDir := t.TempDir()

	var lines []string
	for i := 1; i <= 10; i++ {
		lines = append(lines, fmt.Sprintf("line%d", i))
	}
	testutil.CreateTempFile(t, tmpDir, "startonly.txt", strings.Join(lines, "\n"))

	output := ExecuteReadFile(filepath.Join(tmpDir, "startonly.txt"), 5, 0)

	if !strings.Contains(output, "5: line5") {
		t.Errorf("Expected output to start at line 5, got: %s", output)
	}
	if strings.Contains(output, "4: line4") {
		t.Errorf("Output should NOT contain line 4, got: %s", output)
	}
	if !strings.Contains(output, "10: line10") {
		t.Errorf("Expected output to contain last line, got: %s", output)
	}
}

func TestExecuteReadFile_EndLineOnly(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()
	tmpDir := t.TempDir()

	var lines []string
	for i := 1; i <= 10; i++ {
		lines = append(lines, fmt.Sprintf("line%d", i))
	}
	testutil.CreateTempFile(t, tmpDir, "endonly.txt", strings.Join(lines, "\n"))

	output := ExecuteReadFile(filepath.Join(tmpDir, "endonly.txt"), 0, 3)

	if !strings.Contains(output, "1: line1") {
		t.Errorf("Expected output to start at line 1, got: %s", output)
	}
	if !strings.Contains(output, "3: line3") {
		t.Errorf("Expected output to contain line 3, got: %s", output)
	}
	if strings.Contains(output, "4: line4") {
		t.Errorf("Output should NOT contain line 4, got: %s", output)
	}
}

func TestExecuteReadFile_StartLineLargeFile(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()
	tmpDir := t.TempDir()

	var lines []string
	for i := 1; i <= 300; i++ {
		lines = append(lines, fmt.Sprintf("content_line_%d", i))
	}
	testutil.CreateTempFile(t, tmpDir, "large_start.txt", strings.Join(lines, "\n"))

	output := ExecuteReadFile(filepath.Join(tmpDir, "large_start.txt"), 250, 0)

	if !strings.Contains(output, "250: content_line_250") {
		t.Errorf("Expected output to start at line 250, got: %s", output)
	}
	if strings.Contains(output, "249: content_line_249") {
		t.Errorf("Output should NOT contain line 249, got: %s", output)
	}
	if !strings.Contains(output, "300: content_line_300") {
		t.Errorf("Expected output to contain last line 300, got: %s", output)
	}
}

func TestExecuteReadFile_StartLineOnly_DefaultMaxReadLines(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()
	tmpDir := t.TempDir()

	var lines []string
	for i := 1; i <= 1500; i++ {
		lines = append(lines, fmt.Sprintf("line%d", i))
	}
	testutil.CreateTempFile(t, tmpDir, "window.txt", strings.Join(lines, "\n"))

	output := ExecuteReadFile(filepath.Join(tmpDir, "window.txt"), 100, 0)

	if !strings.Contains(output, "100: line100") {
		t.Fatalf("expected output to start at line 100, got: %s", output)
	}
	if !strings.Contains(output, "1099: line1099") {
		t.Fatalf("expected output to include the 1000-line default window, got: %s", output)
	}
	if strings.Contains(output, "1100: line1100") {
		t.Fatalf("expected output to stop at line 1099 when end_line is omitted, got: %s", output)
	}
}

func TestExecuteReadFile_TrailingNewlineDoesNotCreateExtraLine(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "trailing.txt", "line1\nline2\nline3\n")

	output := ExecuteReadFile(filepath.Join(tmpDir, "trailing.txt"), 4, 4)

	if !strings.Contains(output, "Error: start_line 4 exceeds total lines 3") {
		t.Fatalf("expected EOF validation against 3 real lines, got: %s", output)
	}
	if strings.Contains(output, "4: ") || strings.Contains(output, "of 4") {
		t.Fatalf("trailing newline must not create a phantom fourth line, got: %s", output)
	}
}
