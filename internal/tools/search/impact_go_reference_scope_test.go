package search

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStructuredGoImpactFallbackReferenceSearchPath_DerivesScopedGlobPath(t *testing.T) {
	root := t.TempDir()
	opts := newGoImpactWorkspaceSearchOptions(
		root,
		root,
		root,
		"Run",
		"packages/app/src/**/*.go",
	)

	got := structuredGoImpactFallbackReferenceSearchPath(opts)
	want := filepath.Join(root, "packages", "app", "src")
	if got != want {
		t.Fatalf("reference search path = %q, want %q", got, want)
	}
}

func TestStructuredGoImpactFallbackReferenceSearchPath_DerivesRelativeScopedGlobFromInvocationCWD(t *testing.T) {
	root := t.TempDir()
	cwd := filepath.Join(root, "pkg")
	opts := newGoImpactWorkspaceSearchOptions(root, cwd, ".", "Run", "src/**/*.go")

	got := structuredGoImpactFallbackReferenceSearchPath(opts)
	want := filepath.Join(cwd, "src")
	if got != want {
		t.Fatalf("reference search path = %q, want %q", got, want)
	}
}

func TestStructuredGoImpactFallbackReferenceSearchPath_WidensDirectFileToWorkspace(t *testing.T) {
	root := t.TempDir()
	filePath := filepath.Join(root, "pkg", "run.go")
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filePath, []byte("package pkg\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := structuredGoImpactFallbackReferenceSearchPath(SearchOptions{
		Path:               filePath,
		ProjectMapRootPath: root,
		InvocationCWD:      root,
	})
	if got != root {
		t.Fatalf("reference search path = %q, want workspace root %q", got, root)
	}
}
