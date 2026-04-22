package file

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/testutil"
)

func TestExecuteStrReplace_MultipleMatches_SummaryFirst(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testutil.CreateTempFile(t, tmpDir, "test.txt", "foo\nalpha\nfoo\nbeta\nfoo")

	output, err := executeStrReplaceForTest(testFile, "foo", "REPLACED", "", "")
	if err != nil {
		t.Fatalf("ExecuteStrReplace should not return error: %v", err)
	}
	if !strings.Contains(output, "Candidates: 3 total (showing 2)") {
		t.Fatalf("expected compact candidate summary, got: %s", output)
	}
	if !strings.Contains(output, "- ... 1 more candidates") {
		t.Fatalf("expected omitted candidate summary, got: %s", output)
	}
	if strings.Contains(output, "File preview (first") {
		t.Fatalf("did not expect verbose file preview, got: %s", output)
	}
	if strings.Contains(output, "Next actions:") {
		t.Fatalf("did not expect verbose numbered next actions, got: %s", output)
	}
}

func TestExecuteStrReplace_NotFound_SummaryFirst(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testutil.CreateTempFile(t, tmpDir, "test.txt", "first\nsecond\nthird\nfourth")

	output, err := executeStrReplaceForTest(testFile, "missing", "REPLACED", "", "")
	if err != nil {
		t.Fatalf("ExecuteStrReplace should not return error: %v", err)
	}
	if !strings.Contains(output, "Preview: 1:first | 2:second | 3:third | ... +1 more lines") {
		t.Fatalf("expected compact preview, got: %s", output)
	}
	if !strings.Contains(output, "Next: use read_file/search_code to copy the exact text") {
		t.Fatalf("expected concise next action, got: %s", output)
	}
	if strings.Contains(output, "1)") {
		t.Fatalf("did not expect numbered next actions, got: %s", output)
	}
}
