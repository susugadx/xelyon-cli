package mutation

import (
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/testutil"
)

func TestExecuteStrReplace_GoSyntaxWarning(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()
	setupTestConfirm(t, true)

	var stdout strings.Builder

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "main.go")
	testutil.CreateTempFile(t, tmpDir, "main.go", "package main\n\nfunc Build() error {\n\treturn nil\n}\n")

	result, err := executeStrReplaceWithWritersForTest(&stdout, io.Discard, testFile, "func Build() error {", "func Build() error ", "", "")
	if err != nil {
		t.Fatalf("ExecuteStrReplace failed: %v", err)
	}
	if !strings.Contains(result, "AST syntax check found issues after replacement") {
		t.Fatalf("expected syntax warning in result, got: %s", result)
	}
	if !strings.Contains(stdout.String(), "AST syntax check found issues after replacement") {
		t.Fatalf("expected syntax warning on stdout, got: %s", stdout.String())
	}
	testutil.AssertFileContent(t, testFile, "package main\n\nfunc Build() error \n\treturn nil\n}\n")
}

func TestExecuteStrReplace_GoSyntaxValid(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()
	setupTestConfirm(t, true)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "main.go")
	testutil.CreateTempFile(t, tmpDir, "main.go", "package main\n\nfunc Build() error {\n\treturn nil\n}\n")

	result, err := executeStrReplaceForTest(testFile, "return nil", "panic(\"boom\")", "", "")
	if err != nil {
		t.Fatalf("ExecuteStrReplace failed: %v", err)
	}
	if strings.Contains(result, "AST syntax check found issues after replacement") {
		t.Fatalf("did not expect syntax warning, got: %s", result)
	}
	testutil.AssertFileContent(t, testFile, "package main\n\nfunc Build() error {\n\tpanic(\"boom\")\n}\n")
}

func TestExecuteStrReplace_NonGoNoValidation(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()
	setupTestConfirm(t, true)

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "main.py")
	testutil.CreateTempFile(t, tmpDir, "main.py", "def main():\n    print('ok')\n")

	result, err := executeStrReplaceForTest(testFile, "print('ok')", "if (", "", "")
	if err != nil {
		t.Fatalf("ExecuteStrReplace failed: %v", err)
	}
	if strings.Contains(result, "AST syntax check found issues after replacement") {
		t.Fatalf("did not expect syntax warning, got: %s", result)
	}
	testutil.AssertFileContent(t, testFile, "def main():\n    if (\n")
}
