package navigation

import (
	"context"
	"errors"
	"path/filepath"
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

func TestResolveInspectSymbolAuto_FiltersLSPReferencesBeforeClassification(t *testing.T) {
	setupTestGoFiles(t, map[string]string{
		"run.go": `package example

func Run() {
}
`,
		"app/caller.go": `package app

func main() {
	Run()
}
`,
		"other/caller.go": `package other

func main() {
	Run()
}
`,
	})

	client := &mockNavigationLSPClient{
		refs: []LSPLocation{
			{File: "other/caller.go", Line: 4, Character: 1, EndLine: 4, EndChar: 5},
			{File: "app/caller.go", Line: 4, Character: 1, EndLine: 4, EndChar: 5},
		},
	}

	result, _, status := ResolveInspectSymbolAuto("Run", "", InspectSymbolAutoOptions{
		Budget:    FullBudget,
		LSPClient: client,
		ReferenceFilter: func(ref Reference) bool {
			return strings.Contains(filepath.ToSlash(ref.ResolvedPath), "/app/")
		},
	})

	if status != SymbolAutoSingle {
		t.Fatalf("expected SymbolAutoSingle, got %s", status)
	}
	if len(result.Callers) != 1 {
		t.Fatalf("callers = %+v, want one in-scope LSP caller", result.Callers)
	}
	if result.Callers[0].File != "app/caller.go" {
		t.Fatalf("caller file = %q, want app/caller.go", result.Callers[0].File)
	}
	if result.ReferenceDiagnostics.ResolvedBy != "lsp" {
		t.Fatalf("ResolvedBy = %q, want lsp", result.ReferenceDiagnostics.ResolvedBy)
	}
	if !result.ReferenceDiagnostics.LSPAttempted || !result.ReferenceDiagnostics.LSPAvailable {
		t.Fatalf("LSP diagnostics = %+v, want attempted and available", result.ReferenceDiagnostics)
	}
	if result.ReferenceDiagnostics.RawRefCount != 2 || result.ReferenceDiagnostics.AcceptedRefCount != 1 || result.ReferenceDiagnostics.DroppedRefCount != 1 {
		t.Fatalf("ref counts = raw %d accepted %d dropped %d, want 2/1/1", result.ReferenceDiagnostics.RawRefCount, result.ReferenceDiagnostics.AcceptedRefCount, result.ReferenceDiagnostics.DroppedRefCount)
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

	result, _, status := ResolveInspectSymbolAuto("Run", "", InspectSymbolAutoOptions{
		Budget:    FullBudget,
		LSPClient: client,
	})
	if status != SymbolAutoSingle {
		t.Fatalf("expected SymbolAutoSingle from resolved inspect, got %s", status)
	}
	if result.ReferenceDiagnostics.ResolvedBy != "mixed" {
		t.Fatalf("ResolvedBy = %q, want mixed", result.ReferenceDiagnostics.ResolvedBy)
	}
	if !result.ReferenceDiagnostics.LSPAttempted || !result.ReferenceDiagnostics.FallbackUsed {
		t.Fatalf("LSP fallback diagnostics = %+v, want attempted fallback", result.ReferenceDiagnostics)
	}
	if result.ReferenceDiagnostics.FallbackReason != "lsp_error" {
		t.Fatalf("FallbackReason = %q, want lsp_error", result.ReferenceDiagnostics.FallbackReason)
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
