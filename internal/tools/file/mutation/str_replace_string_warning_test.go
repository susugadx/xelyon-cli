package mutation

import (
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/testutil"
)

func TestExecuteStrReplace_DuplicateWarningForNearbyMatch(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()
	setupTestConfirm(t, true)

	var output strings.Builder

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testutil.CreateTempFile(t, tmpDir, "test.txt", "alpha\nEXISTING\nomega")

	replaceOutput, err := executeStrReplaceWithWritersForTest(&output, io.Discard, testFile, "alpha", "EXISTING", "", "")
	if err != nil {
		t.Fatalf("ExecuteStrReplace failed: %v", err)
	}
	if !strings.Contains(output.String(), "Warning: new_str already exists near the replacement") {
		t.Errorf("Expected warning output, got: %s", output.String())
	}
	if !strings.Contains(replaceOutput, "Successfully replaced") {
		t.Errorf("Expected success message, got: %s", replaceOutput)
	}
	testutil.AssertFileContent(t, testFile, "EXISTING\nEXISTING\nomega")
}

func TestExecuteStrReplace_StringReplace_NoWarningWhenUnique(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()
	setupTestConfirm(t, true)

	var output strings.Builder

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testutil.CreateTempFile(t, tmpDir, "test.txt", "alpha\nEXISTING\nomega")

	replaceOutput, err := executeStrReplaceWithWritersForTest(&output, io.Discard, testFile, "alpha", "NEWVALUE", "", "")
	if err != nil {
		t.Fatalf("ExecuteStrReplace failed: %v", err)
	}
	if strings.Contains(output.String(), "Warning: new_str already exists near the replacement") {
		t.Errorf("Did not expect warning output, got: %s", output.String())
	}
	if !strings.Contains(replaceOutput, "Successfully replaced") {
		t.Errorf("Expected success message, got: %s", replaceOutput)
	}
	testutil.AssertFileContent(t, testFile, "NEWVALUE\nEXISTING\nomega")
}

func TestExecuteStrReplace_NoDuplicateWarningForDistantMatch(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()
	setupTestConfirm(t, true)

	var output strings.Builder

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	lines := []string{"TARGET"}
	for i := 0; i < 20; i++ {
		lines = append(lines, fmt.Sprintf("middle-%d", i))
	}
	lines = append(lines, "EXISTING")
	testutil.CreateTempFile(t, tmpDir, "test.txt", strings.Join(lines, "\n"))

	result, err := executeStrReplaceWithWritersForTest(&output, io.Discard, testFile, "TARGET", "EXISTING", "", "")
	if err != nil {
		t.Fatalf("ExecuteStrReplace failed: %v", err)
	}
	if strings.Contains(output.String(), "Warning: new_str already exists near the replacement") {
		t.Fatalf("did not expect nearby warning, got: %s", output.String())
	}
	if !strings.Contains(result, "Successfully replaced") {
		t.Fatalf("expected success message, got: %s", result)
	}
}
