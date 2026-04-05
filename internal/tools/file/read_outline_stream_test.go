package file

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteReadFile_LargeFileStreaming_Outline(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()
	tmpDir := t.TempDir()
	largeFile := filepath.Join(tmpDir, "large.log")

	f, err := os.Create(largeFile)
	if err != nil {
		t.Fatalf("failed to create large file: %v", err)
	}
	for i := 0; i < 25000; i++ {
		if _, err := fmt.Fprintf(f, "line %05d: %s\n", i+1, strings.Repeat("x", 70)); err != nil {
			_ = f.Close()
			t.Fatalf("failed to write large file: %v", err)
		}
	}
	if err := f.Close(); err != nil {
		t.Fatalf("failed to close large file: %v", err)
	}

	output := ExecuteReadFile(largeFile, 0, 0)
	if !strings.Contains(output, `lines total. For specific sections: paths=["`) {
		t.Fatalf("expected outline footer for large file, got: %s", output)
	}
	if !strings.Contains(output, "1: line 00001") {
		t.Fatalf("expected first line in output, got: %s", output)
	}
	if !strings.Contains(output, "Last lines") {
		t.Fatalf("expected 'Last lines' section, got: %s", output)
	}
}

func TestExecuteReadFile_LargeFileStreaming_GoOutline(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()
	tmpDir := t.TempDir()
	largeFile := filepath.Join(tmpDir, "large.go")

	f, err := os.Create(largeFile)
	if err != nil {
		t.Fatalf("failed to create large file: %v", err)
	}
	fmt.Fprintf(f, "package main\n\nimport \"fmt\"\n\n")
	for i := 0; i < 500; i++ {
		fmt.Fprintf(f, "func handler%d() {\n", i)
		for j := 0; j < 30; j++ {
			fmt.Fprintf(f, "\tfmt.Println(%q)\n", strings.Repeat("x", 100))
		}
		fmt.Fprintf(f, "}\n\n")
	}
	if err := f.Close(); err != nil {
		t.Fatalf("failed to close large file: %v", err)
	}

	output := ExecuteReadFile(largeFile, 0, 0)

	if !strings.Contains(output, "Signatures") {
		t.Fatalf("expected 'Signatures' section for Go file, got: %s", output)
	}
	if !strings.Contains(output, "func handler") {
		t.Fatalf("expected function signatures, got: %s", output)
	}
	if !strings.Contains(output, `lines total. For specific sections: paths=["`) {
		t.Fatalf("expected outline footer, got: %s", output)
	}
}
