package repomap

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuild_CorruptCacheFallsBackToFreshBuild(t *testing.T) {
	requireRipgrep(t)
	setProjectMapTestHome(t)

	root := t.TempDir()
	writeProjectMapTestFile(t, root, "main.go", "package main\n\nfunc Build() error {\n\treturn nil\n}\n")

	cachePath, err := cacheFilePath(root)
	if err != nil {
		t.Fatalf("cacheFilePath() error = %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(cachePath), 0755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}
	if err := os.WriteFile(cachePath, []byte("{invalid"), 0600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}

	pm := NewProjectMap(root, 4000)
	if err := pm.Build(); err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	file := findFileEntry(t, pm, "main.go")
	symbol := findSymbol(t, file, "Build")
	if symbol.Kind != "function" {
		t.Fatalf("symbol kind = %q, want function", symbol.Kind)
	}
}
