package refactor

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewRefactorer(t *testing.T) {
	r := NewRefactorer()

	if r.Config.MaxFileLines != 300 {
		t.Errorf("MaxFileLines = %d, want 300", r.Config.MaxFileLines)
	}
	if r.Config.MaxFunctionLines != 50 {
		t.Errorf("MaxFunctionLines = %d, want 50", r.Config.MaxFunctionLines)
	}
	if r.Config.MinDuplicateLines != 10 {
		t.Errorf("MinDuplicateLines = %d, want 10", r.Config.MinDuplicateLines)
	}
}

func TestNewRefactorerWithConfig(t *testing.T) {
	config := RefactorConfig{
		MaxFileLines:      500,
		MaxFunctionLines:  100,
		MinDuplicateLines: 5,
	}
	r := NewRefactorerWithConfig(config)

	if r.Config.MaxFileLines != 500 {
		t.Errorf("MaxFileLines = %d, want 500", r.Config.MaxFileLines)
	}
}

func TestRefactorer_Analyze_EmptyPaths(t *testing.T) {
	r := NewRefactorer()
	_, err := r.Analyze([]string{})

	if err == nil {
		t.Error("expected error for empty paths")
	}
}

func TestRefactorer_Analyze_InvalidPath(t *testing.T) {
	r := NewRefactorer()
	_, err := r.Analyze([]string{"/nonexistent/path"})

	if err == nil {
		t.Error("expected error for invalid path")
	}
}

func TestRefactorer_Analyze_ValidFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")

	content := `package main

func main() {
	println("hello")
}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	r := NewRefactorer()
	report, err := r.Analyze([]string{testFile})

	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if report.Stats.FilesAnalyzed != 1 {
		t.Errorf("FilesAnalyzed = %d, want 1", report.Stats.FilesAnalyzed)
	}
}

func TestRefactorer_Analyze_LargeFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "large.go")

	// Create a file with more than 300 lines
	var content string
	content = "package main\n\n"
	for i := 0; i < 350; i++ {
		content += "func f" + string(rune('a'+i%26)) + "() {}\n"
	}

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	r := NewRefactorer()
	report, err := r.Analyze([]string{testFile})

	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	if report.Stats.LargeFiles == 0 {
		t.Error("expected at least one large file proposal")
	}
}

func TestRefactorer_Analyze_Directory(t *testing.T) {
	tmpDir := t.TempDir()

	// Create multiple files
	files := []string{"a.go", "b.go", "c.py"}
	for _, f := range files {
		path := filepath.Join(tmpDir, f)
		os.WriteFile(path, []byte("package main\nfunc main() {}\n"), 0644)
	}

	r := NewRefactorer()
	report, err := r.Analyze([]string{tmpDir})

	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	// Should find 3 source files
	if report.Stats.FilesAnalyzed != 3 {
		t.Errorf("FilesAnalyzed = %d, want 3", report.Stats.FilesAnalyzed)
	}
}

func TestRefactorer_Analyze_GlobPattern(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	os.WriteFile(filepath.Join(tmpDir, "a.go"), []byte("package a"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "b.go"), []byte("package b"), 0644)
	os.WriteFile(filepath.Join(tmpDir, "c.txt"), []byte("not source"), 0644)

	r := NewRefactorer()
	report, err := r.Analyze([]string{filepath.Join(tmpDir, "*.go")})

	if err != nil {
		t.Fatalf("Analyze() error = %v", err)
	}

	// Should find only .go files
	if report.Stats.FilesAnalyzed != 2 {
		t.Errorf("FilesAnalyzed = %d, want 2", report.Stats.FilesAnalyzed)
	}
}

func TestFilterByType(t *testing.T) {
	proposals := []RefactorProposal{
		{Type: RefactorSplitFile},
		{Type: RefactorExtractMethod},
		{Type: RefactorSplitFile},
		{Type: RefactorDRY},
	}

	filtered := FilterByType(proposals, RefactorSplitFile)

	if len(filtered) != 2 {
		t.Errorf("len(filtered) = %d, want 2", len(filtered))
	}

	for _, p := range filtered {
		if p.Type != RefactorSplitFile {
			t.Errorf("filtered proposal has type %s, want %s", p.Type, RefactorSplitFile)
		}
	}
}

func TestFilterByType_EmptyFilter(t *testing.T) {
	proposals := []RefactorProposal{
		{Type: RefactorSplitFile},
		{Type: RefactorExtractMethod},
	}

	filtered := FilterByType(proposals, "")

	if len(filtered) != len(proposals) {
		t.Errorf("empty filter should return all proposals")
	}
}

func TestIsSourceFile(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"main.go", true},
		{"script.py", true},
		{"app.js", true},
		{"component.ts", true},
		{"Component.tsx", true},
		{"Main.java", true},
		{"lib.rs", true},
		{"app.rb", true},
		{"index.php", true},
		{"main.c", true},
		{"util.h", true},
		{"class.cpp", true},
		{"header.hpp", true},
		{"README.md", false},
		{"config.yaml", false},
		{"data.json", false},
		{"image.png", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := isSourceFile(tt.path)
			if got != tt.want {
				t.Errorf("isSourceFile(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.MaxFileLines != 300 {
		t.Errorf("MaxFileLines = %d, want 300", config.MaxFileLines)
	}
	if config.MaxFunctionLines != 50 {
		t.Errorf("MaxFunctionLines = %d, want 50", config.MaxFunctionLines)
	}
	if config.MinDuplicateLines != 10 {
		t.Errorf("MinDuplicateLines = %d, want 10", config.MinDuplicateLines)
	}
	if config.UseAI != false {
		t.Error("UseAI should be false by default")
	}
}
