package navigation

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

func TestInspectSymbolAuto_Empty(t *testing.T) {
	output, status := InspectSymbolAuto("", "", nil, nil)
	if status != SymbolAutoNone {
		t.Fatalf("expected SymbolAutoNone, got %s", status)
	}
	if output != "" {
		t.Errorf("expected empty output, got: %s", output)
	}
}

func TestInspectSymbolAuto_NotFound(t *testing.T) {
	if !common.IsRipgrepAvailable() {
		t.Skip("ripgrep not available")
	}
	setupTestGoFile(t, "example.go", testGoSource)

	output, status := InspectSymbolAuto("NonExistentXYZ12345", "", nil, nil)
	if status != SymbolAutoNone {
		t.Fatalf("expected SymbolAutoNone, got %s", status)
	}
	if output != "" {
		t.Errorf("expected empty output, got: %s", output)
	}
}

func TestInspectSymbolAuto_SingleCandidate(t *testing.T) {
	if !common.IsRipgrepAvailable() {
		t.Skip("ripgrep not available")
	}
	setupTestGoFile(t, "example.go", testGoSource)

	output, status := InspectSymbolAuto("Run", "", nil, nil)
	if status != SymbolAutoSingle {
		t.Fatalf("expected SymbolAutoSingle, got %s", status)
	}
	if !strings.Contains(output, "Run") {
		t.Error("expected output to contain symbol name")
	}
	if !strings.Contains(output, "func") {
		t.Error("expected output to contain definition kind")
	}
}

func TestInspectSymbolAuto_MultipleCandidates(t *testing.T) {
	if !common.IsRipgrepAvailable() {
		t.Skip("ripgrep not available")
	}
	setupTestGoFile(t, "example.go", testGoSource)

	output, status := InspectSymbolAuto("Build", "", nil, nil)
	if status != SymbolAutoMultiple {
		t.Fatalf("expected SymbolAutoMultiple, got %s", status)
	}
	if !strings.Contains(output, "Multiple symbols matched") {
		t.Error("expected 'Multiple symbols matched' in output")
	}
}
