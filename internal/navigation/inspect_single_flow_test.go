package navigation

import (
	"strings"
	"testing"
)

func TestInspectSymbol_EmptySymbol(t *testing.T) {
	result := InspectSymbol("", "", "")
	if !strings.Contains(result, "Error") {
		t.Errorf("expected error for empty symbol, got: %s", result)
	}
}

func TestInspectSymbol_NotFound(t *testing.T) {
	result := InspectSymbol("NonExistentSymbol_XYZ_12345", "", "")
	if !strings.Contains(result, "No symbol found") {
		t.Errorf("expected 'No symbol found', got: %s", result)
	}
}

func TestInspectSymbol_SingleCandidate(t *testing.T) {
	setupTestGoFile(t, "example.go", testGoSource)

	result := InspectSymbol("Run", "", "")
	if result == "" {
		t.Fatal("expected non-empty result")
	}
	if strings.Contains(result, "No symbol found") {
		t.Fatalf("expected to find symbol, got: %s", result)
	}
	if !strings.Contains(result, "Run") {
		t.Errorf("expected Run in output, got: %s", result)
	}
	if !strings.Contains(result, "func Run") {
		t.Errorf("expected function definition in output, got: %s", result)
	}
}

func TestInspectSymbol_FullMode(t *testing.T) {
	setupTestGoFile(t, "example.go", testGoSource)

	result := InspectSymbol("Config", "", "full")
	if result == "" {
		t.Fatal("expected non-empty result")
	}
	if strings.Contains(result, "No symbol found") {
		t.Fatalf("expected to find symbol, got: %s", result)
	}
	if !strings.Contains(result, "Config") {
		t.Errorf("expected Config in output, got: %s", result)
	}
}

func TestInspectSymbol_WithCallers(t *testing.T) {
	setupTestGoFile(t, "example.go", testGoSource)

	// Build を Run から呼んでいるので caller がある
	// ただし同名のメソッド Build もあるため複数候補になる可能性がある
	result := InspectSymbol("Run", "", "")
	if strings.Contains(result, "No symbol found") {
		t.Fatalf("expected to find symbol, got: %s", result)
	}
	// Run は単一候補のはず
	if strings.Contains(result, "Multiple symbols matched") {
		t.Errorf("expected single candidate for Run, got: %s", result)
	}
}

func TestInspectSymbol_WithTests(t *testing.T) {
	setupTestGoFiles(t, map[string]string{
		"example.go": `package example

func Greet() string {
	return "hello"
}
`,
		"example_test.go": `package example

import "testing"

func TestGreet(t *testing.T) {
	if Greet() != "hello" {
		t.Error("wrong")
	}
}
`,
	})

	result := InspectSymbol("Greet", "", "")
	if strings.Contains(result, "No symbol found") {
		t.Fatalf("expected to find symbol, got: %s", result)
	}
	if !strings.Contains(result, "Related tests") {
		t.Errorf("expected related tests section, got: %s", result)
	}
	if !strings.Contains(result, "TestGreet") {
		t.Errorf("expected TestGreet in output, got: %s", result)
	}
	if strings.Contains(result, "[test]") {
		t.Errorf("test refs should be shown only in Related tests, got: %s", result)
	}
	if strings.Contains(result, "References (") || strings.Contains(result, "References:") {
		t.Errorf("test-only references must not appear in References section, got: %s", result)
	}
}
