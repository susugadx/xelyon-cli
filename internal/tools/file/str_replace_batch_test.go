package file

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/testutil"
)

func TestExecuteBatchEdits_NormalizedWhitespaceFallback(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	originalContent := "func main() {\n\tfmt.Println(\"hello\")\n}"
	testutil.CreateTempFile(t, tmpDir, "test.txt", originalContent)

	editsJSON := `[{"old_str":"func main() {\n    fmt.Println(\"hello\")\n}","new_str":"func main() {\n\tfmt.Println(\"world\")\n}"}]`
	output, err := executeBatchEditsForTest(testFile, editsJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "Successfully applied 1 edits") && !strings.Contains(output, "Summary") {
		t.Errorf("expected batch success message, got: %s", output)
	}
	testutil.AssertFileContent(t, testFile, "func main() {\n\tfmt.Println(\"world\")\n}")
}

func TestExecuteBatchEdits_Success(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testutil.CreateTempFile(t, tmpDir, "test.txt", "aaa\nbbb\nccc")

	editsJSON := `[{"old_str":"aaa","new_str":"AAA"},{"old_str":"ccc","new_str":"CCC"}]`
	output, err := executeBatchEditsForTest(testFile, editsJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "Successfully applied 2 edits") {
		t.Errorf("expected batch success message, got: %s", output)
	}
	testutil.AssertFileContent(t, testFile, "AAA\nbbb\nCCC")
}

func TestExecuteBatchEdits_StatsReflectEditedLinesNotWholeFile(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")

	lines := make([]string, 200)
	for i := range lines {
		lines[i] = fmt.Sprintf("line %d", i+1)
	}
	lines[99] = "TARGET"
	testutil.CreateTempFile(t, tmpDir, "test.txt", strings.Join(lines, "\n"))

	editsJSON := `[{"old_str":"TARGET","new_str":"UPDATED"}]`

	var stdout bytes.Buffer
	output, err := executeBatchEditsWithPromptIOAndOptions(testPromptIO(&stdout, &stdout), testConfirmOptions(), testFile, editsJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "Successfully applied 1 edits") {
		t.Fatalf("expected batch success message, got: %s", output)
	}

	rendered := stdout.String()
	if !strings.Contains(rendered, "-1 / +1 (net 0)") {
		t.Fatalf("expected edited-line stats, got: %q", rendered)
	}
	if strings.Contains(rendered, "-200 / +200") {
		t.Fatalf("expected not to show whole-file stats, got: %q", rendered)
	}
}

func TestExecuteBatchEdits_RollbackOnFailure(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	originalContent := "aaa\nbbb\nccc"
	testutil.CreateTempFile(t, tmpDir, "test.txt", originalContent)

	editsJSON := `[{"old_str":"aaa","new_str":"AAA"},{"old_str":"zzz","new_str":"ZZZ"}]`
	output, err := executeBatchEditsForTest(testFile, editsJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "edits[1].old_str not found") {
		t.Errorf("expected failure on second edit, got: %s", output)
	}
	testutil.AssertFileContent(t, testFile, originalContent)
}

func TestExecuteBatchEdits_AmbiguousMatch(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testutil.CreateTempFile(t, tmpDir, "test.txt", "foo\nbar\nfoo")

	editsJSON := `[{"old_str":"foo","new_str":"baz"}]`
	output, err := executeBatchEditsForTest(testFile, editsJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "appears 2 times") {
		t.Errorf("expected ambiguous match error, got: %s", output)
	}
	testutil.AssertFileContent(t, testFile, "foo\nbar\nfoo")
}

func TestExecuteBatchEdits_EmptyArray(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testutil.CreateTempFile(t, tmpDir, "test.txt", "content")

	output, err := executeBatchEditsForTest(testFile, "[]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "edits array is empty") {
		t.Errorf("expected empty array error, got: %s", output)
	}
}

func TestExecuteBatchEdits_InvalidJSON(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testutil.CreateTempFile(t, tmpDir, "test.txt", "content")

	output, err := executeBatchEditsForTest(testFile, "not-json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "invalid edits JSON") {
		t.Errorf("expected invalid JSON error, got: %s", output)
	}
}

func TestExecuteBatchEdits_PathTraversal(t *testing.T) {
	setupTestEnvironment(t)
	setupTestConfirm(t, true)

	output, err := executeBatchEditsForTest("../../../etc/passwd", `[{"old_str":"old","new_str":"new"}]`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "Error:") {
		t.Errorf("Expected security error for path traversal, got: %s", output)
	}
}

func TestExecuteBatchEdits_OldStrPriority(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testutil.CreateTempFile(t, tmpDir, "test.txt", "aaa\nbbb")

	output, err := executeStrReplaceForTest(testFile, "aaa", "AAA", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "Successfully replaced") {
		t.Errorf("expected single-mode success, got: %s", output)
	}
	testutil.AssertFileContent(t, testFile, "AAA\nbbb")
}

func TestExecuteBatchEdits_UserCancelled(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()
	setupTestConfirm(t, false)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testutil.CreateTempFile(t, tmpDir, "test.txt", "aaa\nbbb")

	editsJSON := `[{"old_str":"aaa","new_str":"AAA"}]`
	output, err := executeBatchEditsForTest(testFile, editsJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "[CANCELLED]") {
		t.Errorf("expected cancellation, got: %s", output)
	}
	testutil.AssertFileContent(t, testFile, "aaa\nbbb")
}

func TestExecuteBatchEdits_GoSyntaxWarning(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()
	setupTestConfirm(t, true)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "main.go")
	testutil.CreateTempFile(t, tmpDir, "main.go", "package main\n\nfunc Build() error {\n\treturn nil\n}\n")

	editsJSON := `[{"old_str":"func Build() error {","new_str":"func Build() error "}]`
	result, err := executeBatchEditsForTest(testFile, editsJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "AST syntax check found issues after replacement") {
		t.Fatalf("expected syntax warning in result, got: %s", result)
	}
	testutil.AssertFileContent(t, testFile, "package main\n\nfunc Build() error \n\treturn nil\n}\n")
}

func TestExecuteBatchEdits_SequentialApplication(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testutil.CreateTempFile(t, tmpDir, "test.txt", "hello world")

	editsJSON := `[{"old_str":"hello","new_str":"hi"},{"old_str":"hi world","new_str":"hi there"}]`
	output, err := executeBatchEditsForTest(testFile, editsJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "Successfully applied 2 edits") {
		t.Errorf("expected success, got: %s", output)
	}
	testutil.AssertFileContent(t, testFile, "hi there")
}

func TestExecuteBatchEdits_AmbiguousMatch_SummaryFirst(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testutil.CreateTempFile(t, tmpDir, "test.txt", "foo\nbar\nfoo\nbaz\nfoo")

	editsJSON := `[{"old_str":"foo","new_str":"baz"}]`
	output, err := executeBatchEditsForTest(testFile, editsJSON)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(output, "Candidates: 3 total (showing 2)") {
		t.Fatalf("expected compact candidate summary, got: %s", output)
	}
	if !strings.Contains(output, "- ... 1 more candidates") {
		t.Fatalf("expected omitted candidate summary, got: %s", output)
	}
	if strings.Contains(output, "Next actions:") {
		t.Fatalf("did not expect verbose numbered next actions, got: %s", output)
	}
}
