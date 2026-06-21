package mutation

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

func TestExecuteWriteFile_CreatePreviewUsesAddOnlyPatchStyle(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

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
	defer withPermissiveValidatePath(t)()

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
	defer withPermissiveValidatePath(t)()

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
	defer withPermissiveValidatePath(t)()

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
