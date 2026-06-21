package readtool

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/testutil"
)

func TestExecuteReadFile_LargeFile_Outline(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()
	tmpDir := t.TempDir()

	var sb strings.Builder
	sb.WriteString("package main\n\nimport \"fmt\"\n\n")
	for i := 0; i < 30; i++ {
		fmt.Fprintf(&sb, "func handler%d() {\n", i)
		for j := 0; j < 70; j++ {
			fmt.Fprintf(&sb, "\tfmt.Println(%d)\n", j)
		}
		sb.WriteString("}\n\n")
	}
	testutil.CreateTempFile(t, tmpDir, "large.go", sb.String())

	output := ExecuteReadFile(filepath.Join(tmpDir, "large.go"), 0, 0)

	if !strings.Contains(output, "Signatures") {
		t.Errorf("Expected outline to contain 'Signatures', got:\n%s", output)
	}
	if !strings.Contains(output, `lines total. For specific sections: paths=["`) {
		t.Errorf("Expected total-lines footer, got:\n%s", output)
	}
	if !strings.Contains(output, "package main") {
		t.Errorf("Expected head lines to contain 'package main'")
	}
	if !strings.Contains(output, "func handler") {
		t.Errorf("Expected signatures to contain function names, got:\n%s", output)
	}
	if !strings.Contains(output, "Last lines") {
		t.Errorf("Expected 'Last lines' section, got:\n%s", output)
	}
}

func TestExecuteReadFile_LargeFile_PlainText(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()
	tmpDir := t.TempDir()

	var lines []string
	for i := 0; i < 2200; i++ {
		lines = append(lines, fmt.Sprintf("log entry %d", i))
	}
	testutil.CreateTempFile(t, tmpDir, "large.txt", strings.Join(lines, "\n"))

	output := ExecuteReadFile(filepath.Join(tmpDir, "large.txt"), 0, 0)

	if !strings.Contains(output, `(2200 lines total. For specific sections: paths=["`) {
		t.Errorf("Expected total-lines footer, got:\n%s", output)
	}
	if !strings.Contains(output, "Last lines") {
		t.Errorf("Expected 'Last lines' section, got:\n%s", output)
	}
	if !strings.Contains(output, "log entry 0") {
		t.Errorf("Expected head to contain first entry")
	}
}

func TestExecuteReadFile_MediumFile_FullContent(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()
	tmpDir := t.TempDir()

	var sb strings.Builder
	sb.WriteString("package main\n\nimport \"fmt\"\n\n")
	for i := 0; i < 3; i++ {
		fmt.Fprintf(&sb, "func process%d() {\n", i)
		for j := 0; j < 40; j++ {
			fmt.Fprintf(&sb, "\tfmt.Println(%d)\n", j)
		}
		sb.WriteString("}\n\n")
	}
	testutil.CreateTempFile(t, tmpDir, "medium.go", sb.String())

	output := ExecuteReadFile(filepath.Join(tmpDir, "medium.go"), 0, 0)

	if strings.Contains(output, "lines total") {
		t.Errorf("Medium file (150 lines) should NOT have outline footer, got:\n%s", output)
	}
	if !strings.Contains(output, "package main") {
		t.Errorf("Expected content to contain 'package main'")
	}
	if !strings.Contains(output, "func process0") {
		t.Errorf("Expected full content to contain function definitions, got:\n%s", output)
	}
}

func TestExecuteReadFile_LargerMediumFile_Outline(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()
	tmpDir := t.TempDir()

	var sb strings.Builder
	sb.WriteString("package main\n\nimport \"fmt\"\n\n")
	for i := 0; i < 45; i++ {
		fmt.Fprintf(&sb, "func process%d() {\n", i)
		for j := 0; j < 45; j++ {
			fmt.Fprintf(&sb, "\tfmt.Println(%d)\n", j)
		}
		sb.WriteString("}\n\n")
	}
	testutil.CreateTempFile(t, tmpDir, "larger_medium.go", sb.String())

	output := ExecuteReadFile(filepath.Join(tmpDir, "larger_medium.go"), 0, 0)

	if !strings.Contains(output, `lines total. For specific sections: paths=["`) {
		t.Errorf("Expected outline footer for large file, got:\n%s", output)
	}
	if !strings.Contains(output, "package main") {
		t.Errorf("Expected head to contain 'package main'")
	}
	if strings.Contains(output, "symbol=") {
		t.Errorf("Outline guide should not mention symbol mode, got:\n%s", output)
	}
}

func TestExecuteReadFile_SmallFile_FullContent(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()
	tmpDir := t.TempDir()

	var lines []string
	for i := 1; i <= 80; i++ {
		lines = append(lines, fmt.Sprintf("line%d", i))
	}
	testutil.CreateTempFile(t, tmpDir, "small.txt", strings.Join(lines, "\n"))

	output := ExecuteReadFile(filepath.Join(tmpDir, "small.txt"), 0, 0)

	if !strings.Contains(output, "1: line1") {
		t.Errorf("Expected full content with line 1")
	}
	if !strings.Contains(output, "80: line80") {
		t.Errorf("Expected full content with line 80")
	}
	if strings.Contains(output, "lines total") {
		t.Errorf("Small file should NOT have outline footer, got:\n%s", output)
	}
}

func TestExecuteReadFile_GoOutline_UsesTotalLinesFooter(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()
	tmpDir := t.TempDir()

	var sb strings.Builder
	sb.WriteString("package main\n\nimport \"fmt\"\n\n")
	for i := 0; i < 45; i++ {
		fmt.Fprintf(&sb, "func handler%d() {\n", i)
		for j := 0; j < 45; j++ {
			fmt.Fprintf(&sb, "\tfmt.Println(%d)\n", j)
		}
		sb.WriteString("}\n\n")
	}
	testutil.CreateTempFile(t, tmpDir, "handlers.go", sb.String())

	output := ExecuteReadFile(filepath.Join(tmpDir, "handlers.go"), 0, 0)

	if !strings.Contains(output, `lines total. For specific sections: paths=["`) {
		t.Errorf("Expected total-lines footer, got:\n%s", output)
	}
	if strings.Contains(output, "Use start_line/end_line") {
		t.Errorf("Outline footer should not nudge another targeted reread, got:\n%s", output)
	}
	if !strings.Contains(output, `paths=["`) {
		t.Errorf("Outline footer should guide parsePath syntax, got:\n%s", output)
	}
	if strings.Contains(output, "symbol=") {
		t.Errorf("Footer should not mention symbol mode, got:\n%s", output)
	}
}

func TestExecuteReadFile_NonGoOutline_NoSymbolHint(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()
	tmpDir := t.TempDir()

	var lines []string
	for i := 0; i < 2200; i++ {
		lines = append(lines, fmt.Sprintf("data line %d", i))
	}
	testutil.CreateTempFile(t, tmpDir, "data.txt", strings.Join(lines, "\n"))

	output := ExecuteReadFile(filepath.Join(tmpDir, "data.txt"), 0, 0)

	if strings.Contains(output, "symbol=") {
		t.Errorf("Non-Go file should NOT have symbol hint, got:\n%s", output)
	}
	if !strings.Contains(output, `(2200 lines total. For specific sections: paths=["`) {
		t.Errorf("Expected outline footer")
	}
}
