package file

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestResolveDirectQuery_MultiFileAndDirectoryClassification(t *testing.T) {
	root := t.TempDir()
	subdir := filepath.Join(root, "pkg")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "b.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	execCtx := tools.ExecutionContext{
		ProjectMapRootPath: root,
		InvocationCWD:      root,
	}

	files, errResult := resolveDirectQuery(execCtx, "a.go,b.go:1")
	if errResult != "" {
		t.Fatalf("expected multi-file direct query to resolve, got %q", errResult)
	}
	if files.Kind != DirectQueryResolutionFiles {
		t.Fatalf("files.Kind = %q, want %q", files.Kind, DirectQueryResolutionFiles)
	}
	if len(files.Targets) != 2 {
		t.Fatalf("len(files.Targets) = %d, want 2", len(files.Targets))
	}

	dir, errResult := resolveDirectQuery(execCtx, "pkg")
	if errResult != "" {
		t.Fatalf("expected directory direct query to resolve, got %q", errResult)
	}
	if dir.Kind != DirectQueryResolutionDirectory {
		t.Fatalf("dir.Kind = %q, want %q", dir.Kind, DirectQueryResolutionDirectory)
	}

	if _, errResult := resolveDirectQuery(execCtx, "a.go,pkg"); errResult == "" {
		t.Fatal("expected mixed file+directory direct query to be rejected")
	}
}

func TestResolveDirectReadTargets_BatchMissingEntryReturnsError(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "sample.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	input, ok := parseDirectQueryInput("sample.go,missing.go")
	if !ok {
		t.Fatal("expected batch direct query to parse")
	}

	_, errResult := resolveDirectReadTargets(tools.ExecutionContext{
		ProjectMapRootPath: root,
		InvocationCWD:      root,
	}, input)
	if errResult == "" {
		t.Fatal("expected strict direct-read resolution to report missing batch entry")
	}
	if errResult != "Error: direct path not found: missing.go" {
		t.Fatalf("errResult = %q, want missing-path direct error", errResult)
	}
}
