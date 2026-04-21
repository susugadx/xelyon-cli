package navigation

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestQueryLSPLocations_UsesCandidatePosition(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "target.go")
	if err := os.WriteFile(file, []byte("package sample\nfunc Build() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cand := SymbolCandidate{
		Name: "Build",
		File: file,
		Line: 2,
	}

	var gotFile string
	var gotLine int
	var gotCol int

	locations, err := queryLSPLocations(cand, func(ctx context.Context, filePath string, line, character int) ([]LSPLocation, error) {
		gotFile = filePath
		gotLine = line
		gotCol = character
		return []LSPLocation{{File: filePath, Line: line, Character: character}}, nil
	})
	if err != nil {
		t.Fatalf("queryLSPLocations() error = %v", err)
	}
	if len(locations) != 1 {
		t.Fatalf("queryLSPLocations() locations = %d, want 1", len(locations))
	}
	if gotFile != file {
		t.Fatalf("query file path = %q, want %q", gotFile, file)
	}
	if gotLine != 2 {
		t.Fatalf("query line = %d, want 2", gotLine)
	}
	if gotCol != 6 {
		t.Fatalf("query character = %d, want 6", gotCol)
	}
}

func TestQueryLSPLocations_PropagatesFindSymbolColumnError(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "target.go")
	if err := os.WriteFile(file, []byte("package sample\nfunc Other() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cand := SymbolCandidate{
		Name: "Build",
		File: file,
		Line: 99,
	}

	called := false
	_, err := queryLSPLocations(cand, func(ctx context.Context, filePath string, line, character int) ([]LSPLocation, error) {
		called = true
		return nil, nil
	})
	if err == nil {
		t.Fatal("queryLSPLocations() error = nil, want non-nil")
	}
	if called {
		t.Fatal("query should not be called when findSymbolColumn fails")
	}
}
