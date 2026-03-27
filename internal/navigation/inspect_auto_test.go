package navigation

import (
	"context"
	"errors"
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

type mockNavigationLSPClient struct {
	refs     []LSPLocation
	refsErr  error
	impls    []LSPLocation
	implsErr error
}

func (m *mockNavigationLSPClient) FindReferences(context.Context, string, int, int, bool) ([]LSPLocation, error) {
	return m.refs, m.refsErr
}

func (m *mockNavigationLSPClient) GotoDefinition(context.Context, string, int, int) ([]LSPLocation, error) {
	return nil, nil
}

func (m *mockNavigationLSPClient) GotoImplementation(context.Context, string, int, int) ([]LSPLocation, error) {
	return m.impls, m.implsErr
}

func TestInspectSymbolAuto_UsesLSPReferences(t *testing.T) {
	setupTestGoFiles(t, map[string]string{
		"run.go": `package example

func Run() {
}
`,
		"caller.go": `package example

func main() {
	Run()
}
`,
	})

	client := &mockNavigationLSPClient{
		refs: []LSPLocation{
			{File: "caller.go", Line: 4, Character: 1, EndLine: 4, EndChar: 5},
		},
	}

	output, status := InspectSymbolAuto("Run", "", nil, client)
	if status != SymbolAutoSingle {
		t.Fatalf("expected SymbolAutoSingle, got %s", status)
	}
	if !strings.Contains(output, "Callers (1)") {
		t.Fatalf("expected callers section from LSP result, got: %s", output)
	}
	if !strings.Contains(output, "resolved via gopls") {
		t.Fatalf("expected gopls suffix, got: %s", output)
	}
}

func TestInspectSymbolAuto_LSPFallbackOnError(t *testing.T) {
	setupTestGoFile(t, "example.go", testGoSource)

	client := &mockNavigationLSPClient{refsErr: errors.New("boom")}

	output, status := InspectSymbolAuto("Run", "", nil, client)
	if status != SymbolAutoSingle {
		t.Fatalf("expected SymbolAutoSingle, got %s", status)
	}
	if strings.Contains(output, "resolved via gopls") {
		t.Fatalf("expected fallback output without gopls suffix, got: %s", output)
	}
	if !strings.Contains(output, "func Run") {
		t.Fatalf("expected fallback output to still inspect the symbol, got: %s", output)
	}
}

func TestInspectSymbolAuto_UsesLSPImplementations(t *testing.T) {
	setupTestGoFile(t, "example.go", `package example

type Builder interface {
	Build() string
}

type FileBuilder struct{}

func (FileBuilder) Build() string { return "" }
`)

	client := &mockNavigationLSPClient{
		refs: []LSPLocation{
			{File: "example.go", Line: 4, Character: 1, EndLine: 4, EndChar: 6},
		},
		impls: []LSPLocation{
			{File: "example.go", Line: 7, Character: 1, EndLine: 7, EndChar: 11},
		},
	}

	output, status := InspectSymbolAuto("Builder", "", nil, client)
	if status != SymbolAutoSingle {
		t.Fatalf("expected SymbolAutoSingle, got %s", status)
	}
	if !strings.Contains(output, "Implementations (1)") {
		t.Fatalf("expected implementations section, got: %s", output)
	}
	if !strings.Contains(output, "resolved via gopls") {
		t.Fatalf("expected gopls suffix, got: %s", output)
	}
}
