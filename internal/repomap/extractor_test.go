//go:build !norepomap
// +build !norepomap

package repomap

import (
	"path/filepath"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/testutil"
)

// ================== Common Tests ==================

func TestExtractUnsupportedLanguage(t *testing.T) {
	content := `Just some text file content`

	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "readme.txt", content)
	testFile := filepath.Join(tmpDir, "readme.txt")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	// サポートされていない言語はnilを返す
	if fileSymbols != nil {
		t.Errorf("Expected nil for unsupported language, got %v", fileSymbols)
	}
}

func TestIsSupportedFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"test.go", true},
		{"test.js", true},
		{"test.ts", true},
		{"test.py", true},
		{"test.rs", true},
		{"test.java", true},
		{"test.rb", true},
		{"test.c", true},
		{"test.cpp", true},
		{"test.kt", true},
		{"test.swift", true},
		{"test.cs", true},
		{"test.scala", true},
		{"test.php", true},
		// Issue #59: CSS/SCSS
		{"test.css", true},
		{"test.scss", true},
		// Issue #60: 設定言語
		{"test.yaml", true},
		{"test.yml", true},
		{"test.toml", true},
		{"test.sql", true},
		{"test.sh", true},
		{"test.bash", true},
		{"test.md", true},
		{"test.txt", false},
		{"test.json", false},
	}

	for _, tt := range tests {
		result := IsSupportedFile(tt.path)
		if result != tt.expected {
			t.Errorf("IsSupportedFile(%s) = %v, expected %v", tt.path, result, tt.expected)
		}
	}
}
