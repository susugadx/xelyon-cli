//go:build !norepomap
// +build !norepomap

package repomap

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/testutil"
)

func TestExtractRustFunction(t *testing.T) {
	content := `fn hello_world(name: &str) -> String {
	format!("Hello, {}!", name)
}

pub fn add(a: i32, b: i32) -> i32 {
	a + b
}
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "test.rs", content)
	testFile := filepath.Join(tmpDir, "test.rs")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	if len(fileSymbols.Symbols) < 2 {
		t.Fatalf("Expected at least 2 symbols, got %d", len(fileSymbols.Symbols))
	}

	// hello_world関数
	hello := fileSymbols.Symbols[0]
	if hello.Name != "hello_world" {
		t.Errorf("Expected 'hello_world', got '%s'", hello.Name)
	}
	if hello.Kind != "function" {
		t.Errorf("Expected 'function', got '%s'", hello.Kind)
	}
	if !strings.Contains(hello.Signature, "fn hello_world") {
		t.Errorf("Expected signature to contain 'fn hello_world', got '%s'", hello.Signature)
	}
}

func TestExtractRustStruct(t *testing.T) {
	content := `struct Point {
	x: i32,
	y: i32,
}

enum Color {
	Red,
	Green,
	Blue,
}
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "types.rs", content)
	testFile := filepath.Join(tmpDir, "types.rs")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	if len(fileSymbols.Symbols) < 2 {
		t.Fatalf("Expected at least 2 symbols, got %d", len(fileSymbols.Symbols))
	}

	// Point struct
	point := fileSymbols.Symbols[0]
	if point.Name != "Point" {
		t.Errorf("Expected 'Point', got '%s'", point.Name)
	}
	if point.Kind != "struct" {
		t.Errorf("Expected 'struct', got '%s'", point.Kind)
	}

	// Color enum
	color := fileSymbols.Symbols[1]
	if color.Name != "Color" {
		t.Errorf("Expected 'Color', got '%s'", color.Name)
	}
	if color.Kind != "enum" {
		t.Errorf("Expected 'enum', got '%s'", color.Kind)
	}
}
