package navigation

import (
	"context"
	"errors"
	"strings"
	"testing"
)

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

func TestInspectSymbolAuto_LSPFallbackStillKeepsImplementations(t *testing.T) {
	setupTestGoFile(t, "example.go", `package example

type Builder interface {
	Build() string
}

type FileBuilder struct{}

func (FileBuilder) Build() string { return "" }

func Use(b Builder) string {
	return b.Build()
}
`)

	client := &mockNavigationLSPClient{
		refsErr: errors.New("boom"),
		impls: []LSPLocation{
			{File: "example.go", Line: 7, Character: 1, EndLine: 7, EndChar: 11},
		},
	}

	output, status := InspectSymbolAuto("Builder", "", nil, client)
	if status != SymbolAutoSingle {
		t.Fatalf("expected SymbolAutoSingle, got %s", status)
	}
	if strings.Contains(output, "resolved via gopls") {
		t.Fatalf("expected fallback output without gopls suffix, got: %s", output)
	}
	if !strings.Contains(output, "Implementations (1)") {
		t.Fatalf("expected implementations section on fallback, got: %s", output)
	}
}
