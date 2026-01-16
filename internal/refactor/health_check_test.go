package refactor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckFileHealth_LargeFile(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "large.go")

	// Create a file with 350 lines
	var content strings.Builder
	content.WriteString("package main\n\n")
	for i := 0; i < 350; i++ {
		content.WriteString("// line " + string(rune('a'+i%26)) + "\n")
	}

	if err := os.WriteFile(testFile, []byte(content.String()), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	config := HealthCheckConfig{
		Enabled:       true,
		MaxFileLines:  300,
		CheckFileSize: true,
	}

	result := CheckFileHealth(testFile, config)

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !result.HasWarning {
		t.Error("expected warning for large file")
	}
	if result.FileLines < 300 {
		t.Errorf("expected file lines >= 300, got %d", result.FileLines)
	}
	if len(result.Suggestions) == 0 {
		t.Error("expected suggestions for large file")
	}
}

func TestCheckFileHealth_LongFunction(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "longfunc.go")

	// Create a file with a long function
	var content strings.Builder
	content.WriteString("package main\n\n")
	content.WriteString("func longFunction() {\n")
	for i := 0; i < 60; i++ {
		content.WriteString("\tprintln(\"line\")\n")
	}
	content.WriteString("}\n")

	if err := os.WriteFile(testFile, []byte(content.String()), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	config := HealthCheckConfig{
		Enabled:          true,
		MaxFunctionLines: 50,
		CheckFuncSize:    true,
	}

	result := CheckFileHealth(testFile, config)

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if !result.HasWarning {
		t.Error("expected warning for long function")
	}
	if len(result.LongFunctions) == 0 {
		t.Error("expected long function to be detected")
	}
	if result.LongFunctions[0].Name != "longFunction" {
		t.Errorf("expected function name 'longFunction', got '%s'", result.LongFunctions[0].Name)
	}
}

func TestCheckFileHealth_Disabled(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.go")

	if err := os.WriteFile(testFile, []byte("package main\n"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	config := HealthCheckConfig{
		Enabled: false,
	}

	result := CheckFileHealth(testFile, config)

	if result != nil {
		t.Error("expected nil result when disabled")
	}
}

func TestCheckFileHealth_NoWarning(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "small.go")

	content := `package main

func main() {
	println("hello")
}
`
	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	config := HealthCheckConfig{
		Enabled:          true,
		MaxFileLines:     300,
		MaxFunctionLines: 50,
		CheckFileSize:    true,
		CheckFuncSize:    true,
	}

	result := CheckFileHealth(testFile, config)

	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.HasWarning {
		t.Error("expected no warning for small file")
	}
}

func TestFormatHealthWarning(t *testing.T) {
	result := &HealthCheckResult{
		FilePath:     "test.go",
		HasWarning:   true,
		FileLines:    400,
		MaxFileLines: 300,
		LongFunctions: []LongFunctionInfo{
			{Name: "bigFunc", Lines: 80, LineStart: 10},
		},
		MaxFunctionLines: 50,
		Suggestions: []string{
			"Split the file",
			"Extract methods",
		},
	}

	output := FormatHealthWarning(result)

	if !strings.Contains(output, "test.go") {
		t.Error("expected output to contain file path")
	}
	if !strings.Contains(output, "400") {
		t.Error("expected output to contain file lines")
	}
	if !strings.Contains(output, "bigFunc") {
		t.Error("expected output to contain function name")
	}
	if !strings.Contains(output, "Suggestions") {
		t.Error("expected output to contain suggestions")
	}
}

func TestFormatHealthWarning_NoWarning(t *testing.T) {
	result := &HealthCheckResult{
		HasWarning: false,
	}

	output := FormatHealthWarning(result)

	if output != "" {
		t.Error("expected empty output for no warning")
	}
}

func TestShouldCheckHealth(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"main.go", true},
		{"script.py", true},
		{"app.js", true},
		{"README.md", false},
		{"config.yaml", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := ShouldCheckHealth(tt.path)
			if got != tt.want {
				t.Errorf("ShouldCheckHealth(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestDefaultHealthCheckConfig(t *testing.T) {
	config := DefaultHealthCheckConfig()

	if !config.Enabled {
		t.Error("expected Enabled to be true by default")
	}
	if config.MaxFileLines != 300 {
		t.Errorf("expected MaxFileLines = 300, got %d", config.MaxFileLines)
	}
	if config.MaxFunctionLines != 50 {
		t.Errorf("expected MaxFunctionLines = 50, got %d", config.MaxFunctionLines)
	}
}
