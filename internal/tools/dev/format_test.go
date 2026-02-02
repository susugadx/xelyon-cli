package dev

import (
	"os"
	"strings"
	"testing"
)

func TestResolveFormatPath_Default(t *testing.T) {
	path, err := resolveFormatPath("")
	if err != nil {
		t.Fatalf("resolveFormatPath returned error: %v", err)
	}
	if path == "" {
		t.Fatalf("resolveFormatPath returned empty path")
	}
}

func TestResolveFormatPath_Glob(t *testing.T) {
	path, err := resolveFormatPath("./...")
	if err != nil {
		t.Fatalf("resolveFormatPath returned error: %v", err)
	}
	if !strings.HasSuffix(path, "xelyon-cli") {
		t.Fatalf("resolveFormatPath should resolve to project root, got %s", path)
	}
}

func TestResolveFormatPath_GlobFromSubdir(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(dir+"/child", 0o755); err != nil {
		t.Fatalf("os.MkdirAll returned error: %v", err)
	}
	if err := os.WriteFile(dir+"/go.mod", []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatalf("os.WriteFile returned error: %v", err)
	}

	path, err := resolveFormatPath(dir + "/child/...")
	if err != nil {
		t.Fatalf("resolveFormatPath returned error: %v", err)
	}
	if path != dir {
		t.Fatalf("resolveFormatPath should resolve to go.mod root, got %s", path)
	}
}

func TestResolveFormatPath_SubdirGlob(t *testing.T) {
	path, err := resolveFormatPath("./internal/...")
	if err != nil {
		t.Fatalf("resolveFormatPath returned error: %v", err)
	}
	if !strings.HasSuffix(path, "xelyon-cli") {
		t.Fatalf("resolveFormatPath should resolve to project root, got %s", path)
	}
}

func TestResolveFormatPath_NoGlob(t *testing.T) {
	path, err := resolveFormatPath(".")
	if err != nil {
		t.Fatalf("resolveFormatPath returned error: %v", err)
	}
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("os.Getwd returned error: %v", err)
	}
	if path != cwd {
		t.Fatalf("resolveFormatPath should resolve to cwd, got %s", path)
	}
}
