package search

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestExecuteSearchCode_Success(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.go")
	content := `package main

func main() {
	fmt.Println("Hello, World!")
	// SEARCH_TARGET
	fmt.Println("End")
}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	pattern := "SEARCH_TARGET"
	output := ExecuteSearchCode(pattern, tmpDir)

	if !strings.Contains(output, "SEARCH_TARGET") {
		t.Errorf("ExecuteSearchCode() output = %v, should contain pattern", output)
	}

	if !strings.Contains(output, "test.go") {
		t.Errorf("ExecuteSearchCode() output = %v, should contain filename", output)
	}
}

func TestExecuteSearchCode_EmptyPattern(t *testing.T) {
	tmpDir := t.TempDir()

	output := ExecuteSearchCode("", tmpDir)

	if !strings.Contains(output, "Error:") || !strings.Contains(output, "required") {
		t.Errorf("ExecuteSearchCode() output = %v, should contain error about required pattern", output)
	}
}

func TestExecuteSearchCode_NoMatches(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.go")
	content := "package main\n"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	pattern := "NONEXISTENT_PATTERN_12345"
	output := ExecuteSearchCode(pattern, tmpDir)

	if !strings.Contains(output, "No matches found") {
		t.Errorf("ExecuteSearchCode() output = %v, should contain 'No matches found'", output)
	}
}

func TestExecuteSearchCode_MultipleMatches(t *testing.T) {
	tmpDir := t.TempDir()

	files := []string{"file1.go", "file2.go", "file3.go"}
	for _, file := range files {
		filePath := filepath.Join(tmpDir, file)
		content := "package main\n// TARGET_PATTERN\n"
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", file, err)
		}
	}

	pattern := "TARGET_PATTERN"
	output := ExecuteSearchCode(pattern, tmpDir)

	for _, file := range files {
		if !strings.Contains(output, file) {
			t.Errorf("ExecuteSearchCode() output should contain file %s", file)
		}
	}

	if !strings.Contains(output, "TARGET_PATTERN") {
		t.Error("ExecuteSearchCode() output should contain pattern")
	}
}

func TestExecuteSearchCode_LineNumbers(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.go")
	content := `Line 1
Line 2
Line 3 PATTERN
Line 4
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	pattern := "PATTERN"
	output := ExecuteSearchCode(pattern, tmpDir)

	if !strings.Contains(output, ":") {
		t.Error("ExecuteSearchCode() output should contain line numbers (colon separated)")
	}
}

func TestExecuteSearchCode_SpecialCharactersInPattern(t *testing.T) {
	tmpDir := t.TempDir()

	testFile := filepath.Join(tmpDir, "test.go")
	content := "package main\n// Pattern with special chars: @#$\n"
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	pattern := "@#$"
	output := ExecuteSearchCode(pattern, tmpDir)

	if !strings.Contains(output, "@#$") {
		t.Errorf("ExecuteSearchCode() should find pattern with special characters, got %v", output)
	}
}

func TestExecuteSearchCode_IgnoreBinaryFiles(t *testing.T) {
	tmpDir := t.TempDir()

	binaryFile := filepath.Join(tmpDir, "binary.bin")
	if err := os.WriteFile(binaryFile, []byte{0x00, 0x01, 0x02, 0xFF}, 0644); err != nil {
		t.Fatalf("Failed to create binary file: %v", err)
	}

	textFile := filepath.Join(tmpDir, "text.go")
	if err := os.WriteFile(textFile, []byte("PATTERN\n"), 0644); err != nil {
		t.Fatalf("Failed to create text file: %v", err)
	}

	pattern := "PATTERN"
	output := ExecuteSearchCode(pattern, tmpDir)

	if !strings.Contains(output, "text.go") {
		t.Error("ExecuteSearchCode() should find pattern in text file")
	}
}

func TestExecuteSearchCode_LongOutput(t *testing.T) {
	tmpDir := t.TempDir()

	for i := 0; i < 100; i++ {
		fileName := filepath.Join(tmpDir, fmt.Sprintf("file%d.go", i))
		content := "package main\n// PATTERN\n"
		if err := os.WriteFile(fileName, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	pattern := "PATTERN"
	output := ExecuteSearchCode(pattern, tmpDir)

	if strings.Contains(output, "more matches") {
		lines := strings.Split(output, "\n")
		if len(lines) > 55 {
			t.Errorf("ExecuteSearchCode() output should be truncated, got %d lines", len(lines))
		}
	}
}

func TestExecuteSearchCode_JavaScriptFile(t *testing.T) {
	tmpDir := t.TempDir()

	jsFile := filepath.Join(tmpDir, "test.js")
	content := "console.log('SEARCH_TARGET');\n"
	if err := os.WriteFile(jsFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create JS file: %v", err)
	}

	pattern := "SEARCH_TARGET"
	output := ExecuteSearchCode(pattern, tmpDir)

	if !strings.Contains(output, "test.js") {
		t.Errorf("ExecuteSearchCode() should search JavaScript files, got %v", output)
	}
}

func TestExecuteSearchCode_MarkdownFile(t *testing.T) {
	tmpDir := t.TempDir()

	mdFile := filepath.Join(tmpDir, "README.md")
	content := "# Title\nSEARCH_TARGET\n"
	if err := os.WriteFile(mdFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create MD file: %v", err)
	}

	pattern := "SEARCH_TARGET"
	output := ExecuteSearchCode(pattern, tmpDir)

	if !strings.Contains(output, "README.md") {
		t.Errorf("ExecuteSearchCode() should search Markdown files, got %v", output)
	}
}

func TestExecuteSearchCode_NonexistentDirectory(t *testing.T) {
	pattern := "PATTERN"
	output := ExecuteSearchCode(pattern, "/nonexistent/directory/12345")

	if len(output) == 0 {
		t.Error("ExecuteSearchCode() should return some output for nonexistent directory")
	}
}
