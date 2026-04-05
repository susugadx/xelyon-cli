package file

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/testutil"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

func TestExecuteDeleteFile_PreviewUsesDeletionToneLines(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "preview.txt")
	testutil.CreateTempFile(t, tmpDir, "preview.txt", "line1\nline2")
	var stdout bytes.Buffer

	_, err := ExecuteDeleteFileWithPromptIOAndOptionsAndLSPClient(
		testPromptIO(&stdout, &stdout),
		testConfirmOptions(),
		nil,
		testFile,
	)
	if err != nil {
		t.Fatalf("ExecuteDeleteFile failed: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "Deleting") {
		t.Fatalf("expected patch-style delete preview header, got: %s", output)
	}
	if !strings.Contains(output, "   1 - line1") || !strings.Contains(output, "   2 - line2") {
		t.Fatalf("expected deletion-tone preview lines, got: %s", output)
	}
}

func TestExecuteDeleteFile_PreviewHeaderUsesRealTotalLineCount(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "count.txt")
	testutil.CreateTempFile(t, tmpDir, "count.txt", "line1\nline2\nline3")
	var stdout bytes.Buffer

	_, err := ExecuteDeleteFileWithPromptIOAndOptionsAndLSPClient(
		testPromptIO(&stdout, &stdout),
		testConfirmOptions(),
		nil,
		testFile,
	)
	if err != nil {
		t.Fatalf("ExecuteDeleteFile failed: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "(-3)") {
		t.Fatalf("expected delete preview summary to use the real total line count, got: %s", output)
	}
}

func TestExecuteDeleteFile_PreviewUnderLimitShowsFullBody(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "full.txt")
	testutil.CreateTempFile(t, tmpDir, "full.txt", "line1\nline2\nline3")
	var stdout bytes.Buffer

	_, err := ExecuteDeleteFileWithPromptIOAndOptionsAndLSPClient(
		testPromptIO(&stdout, &stdout),
		testConfirmOptions(),
		nil,
		testFile,
	)
	if err != nil {
		t.Fatalf("ExecuteDeleteFile failed: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "   3 - line3") {
		t.Fatalf("expected full preview body under the cap, got: %s", output)
	}
	if strings.Contains(output, "preview truncated:") {
		t.Fatalf("did not expect truncation notice under the cap, got: %s", output)
	}
	if !strings.Contains(output, "(-3)") {
		t.Fatalf("expected metadata to use the real total line count, got: %s", output)
	}
}

func TestExecuteDeleteFile_PreviewLineCapKeepsRealRemovedCount(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "large.txt")
	var builder strings.Builder
	const maxPreviewLines = 7
	for i := 1; i <= maxPreviewLines+5; i++ {
		if i > 1 {
			builder.WriteByte('\n')
		}
		_, _ = fmt.Fprintf(&builder, "line%d", i)
	}
	testutil.CreateTempFile(t, tmpDir, "large.txt", builder.String())
	var stdout bytes.Buffer
	cfg := config.DefaultConfig()
	cfg.Diff.MaxTotalLines = maxPreviewLines

	_, err := ExecuteDeleteFileWithPromptIOAndOptionsAndLSPClient(
		testPromptIO(&stdout, &stdout),
		common.ConfirmOptions{Config: cfg},
		nil,
		testFile,
	)
	if err != nil {
		t.Fatalf("ExecuteDeleteFile failed: %v", err)
	}

	output := stdout.String()
	if strings.Contains(output, fmt.Sprintf("%4d - line%d", maxPreviewLines+1, maxPreviewLines+1)) {
		t.Fatalf("did not expect preview lines beyond the soft cap, got: %s", output)
	}
	if !strings.Contains(output, fmt.Sprintf("preview truncated: showing first %d of %d lines", maxPreviewLines, maxPreviewLines+5)) {
		t.Fatalf("expected explicit line-cap truncation notice, got: %s", output)
	}
	if !strings.Contains(output, fmt.Sprintf("(-%d)", maxPreviewLines+5)) {
		t.Fatalf("expected Removed count to use the real total line count, got: %s", output)
	}
}

func TestExecuteDeleteFile_PreviewByteCapSignalsTruncation(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "bytes.txt")
	largeLine := strings.Repeat("a", maxFullBodyPreviewBytes/2)
	testutil.CreateTempFile(t, tmpDir, "bytes.txt", largeLine+"\n"+largeLine)
	var stdout bytes.Buffer
	cfg := config.DefaultConfig()
	cfg.Diff.MaxTotalLines = 0

	_, err := ExecuteDeleteFileWithPromptIOAndOptionsAndLSPClient(
		testPromptIO(&stdout, &stdout),
		common.ConfirmOptions{Config: cfg},
		nil,
		testFile,
	)
	if err != nil {
		t.Fatalf("ExecuteDeleteFile failed: %v", err)
	}

	output := stdout.String()
	if !strings.Contains(output, "preview truncated at 64KB") {
		t.Fatalf("expected explicit byte-cap truncation notice, got: %s", output)
	}
	if !strings.Contains(output, "preview truncated: showing first 1 of 2 lines") {
		t.Fatalf("expected line summary for byte-cap truncation, got: %s", output)
	}
	if !strings.Contains(output, "(-2)") {
		t.Fatalf("expected Removed count to keep the real total line count, got: %s", output)
	}
}

func TestExecuteDeleteFile_ZeroValueConfirmOptionsDoesNotPanic(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "zero.txt")
	testutil.CreateTempFile(t, tmpDir, "zero.txt", "line1\nline2")
	var stdout bytes.Buffer

	result, err := ExecuteDeleteFileWithPromptIOAndOptionsAndLSPClient(
		testPromptIO(&stdout, &stdout),
		common.ConfirmOptions{},
		nil,
		testFile,
	)
	if err != nil {
		t.Fatalf("ExecuteDeleteFile should not return error: %v", err)
	}
	if !strings.Contains(result, "✅ Deleted:") {
		t.Fatalf("expected successful delete with zero-value confirm options, got: %s", result)
	}
	if !strings.Contains(stdout.String(), "Deleting") {
		t.Fatalf("expected preview output without panic, got: %s", stdout.String())
	}
}
