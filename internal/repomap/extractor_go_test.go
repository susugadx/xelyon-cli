//go:build !norepomap
// +build !norepomap

package repomap

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/testutil"
)

func TestExtractGoFunction(t *testing.T) {
	content := `package main

func Hello(name string) string {
	return "Hello, " + name
}

func (s *Server) Start(port int) error {
	return nil
}
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "test.go", content)
	testFile := filepath.Join(tmpDir, "test.go")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	if len(fileSymbols.Symbols) != 2 {
		t.Fatalf("Expected 2 symbols, got %d", len(fileSymbols.Symbols))
	}

	// Hello関数の検証
	hello := fileSymbols.Symbols[0]
	if hello.Name != "Hello" {
		t.Errorf("Expected 'Hello', got '%s'", hello.Name)
	}
	if hello.Kind != "function" {
		t.Errorf("Expected 'function', got '%s'", hello.Kind)
	}
	if !strings.Contains(hello.Signature, "func Hello") {
		t.Errorf("Expected signature to contain 'func Hello', got '%s'", hello.Signature)
	}

	// Startメソッドの検証
	start := fileSymbols.Symbols[1]
	if start.Name != "Start" {
		t.Errorf("Expected 'Start', got '%s'", start.Name)
	}
	if start.Kind != "method" {
		t.Errorf("Expected 'method', got '%s'", start.Kind)
	}
}

func TestExtractGoType(t *testing.T) {
	content := `package main

type User struct {
	Name string
	Age  int
}

type Reader interface {
	Read(p []byte) (n int, err error)
}
`
	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "types.go", content)
	testFile := filepath.Join(tmpDir, "types.go")

	fileSymbols, err := ExtractSymbols(testFile)
	if err != nil {
		t.Fatalf("ExtractSymbols failed: %v", err)
	}
	if fileSymbols == nil {
		t.Fatal("Expected fileSymbols, got nil")
	}

	if len(fileSymbols.Symbols) != 2 {
		t.Fatalf("Expected 2 symbols, got %d", len(fileSymbols.Symbols))
	}

	// User struct
	user := fileSymbols.Symbols[0]
	if user.Name != "User" {
		t.Errorf("Expected 'User', got '%s'", user.Name)
	}
	if user.Kind != "struct" {
		t.Errorf("Expected 'struct', got '%s'", user.Kind)
	}

	// Reader interface
	reader := fileSymbols.Symbols[1]
	if reader.Name != "Reader" {
		t.Errorf("Expected 'Reader', got '%s'", reader.Name)
	}
	if reader.Kind != "interface" {
		t.Errorf("Expected 'interface', got '%s'", reader.Kind)
	}
}
