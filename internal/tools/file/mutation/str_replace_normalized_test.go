package mutation

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/testutil"
)

func TestStrReplace_NormalizedWhitespaceMatch(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")
	originalContent := "package main\n\nfunc hello() {\n\tfmt.Println(\"hello\")\n}\n"
	testutil.CreateTempFile(t, tmpDir, "test.go", originalContent)

	result, err := executeStrReplaceForTest(testFile, "func hello() {\n    fmt.Println(\"hello\")\n}", "func hello() {\n\tfmt.Println(\"world\")\n}", "", "")
	if err != nil {
		t.Fatalf("executeStrReplaceForTest failed: %v", err)
	}
	if !strings.Contains(result, "Successfully replaced") {
		t.Errorf("expected success via normalized whitespace matching, got: %s", result)
	}
	if !strings.Contains(result, "normalized whitespace") {
		t.Errorf("expected 'normalized whitespace' in result, got: %s", result)
	}
	testutil.AssertFileContent(t, testFile, "package main\n\nfunc hello() {\n\tfmt.Println(\"world\")\n}\n")
}

func TestStrReplace_NormalizedWhitespaceMatch_ExtraSpaces(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testutil.CreateTempFile(t, tmpDir, "test.txt", "start\n    indented line\nend")

	result, err := executeStrReplaceForTest(testFile, "  indented line", "  replaced line", "", "")
	if err != nil {
		t.Fatalf("executeStrReplaceForTest failed: %v", err)
	}
	if !strings.Contains(result, "Successfully replaced") {
		t.Errorf("expected success via normalized match, got: %s", result)
	}
}
