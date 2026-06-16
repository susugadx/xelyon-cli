package gathercontext

import (
	"path/filepath"
	"testing"
)

func TestGatherContext_ScopedExactFilenameQueryUsesDirectRead(t *testing.T) {
	root := t.TempDir()
	withGatherContextWorkingDir(t, root)

	writeGatherContextFiles(t, map[string]string{
		filepath.Join(root, "target.go"):        "package main\n\nconst selected = \"root\"\n",
		filepath.Join(root, "pkg", "target.go"): "package pkg\n\nconst selected = \"subdir\"\n",
	})

	result, _ := runGatherContext(t, newGatherContextExecCtx(root), map[string]string{
		"query":       "target.go",
		"path":        "pkg",
		"file_filter": "go",
	})
	assertGatherContextContainsAll(t, result, "Route: Direct read", "📄 File: pkg/target.go", `"subdir"`)
	assertGatherContextExcludesAll(t, result, "Route: Direct query", "No matches found", `"root"`)
}

func TestGatherContext_ScopedBareFilenameMissReturnsDirectError(t *testing.T) {
	root := t.TempDir()
	withGatherContextWorkingDir(t, root)

	writeGatherContextFiles(t, map[string]string{
		filepath.Join(root, "go.mod"):          "module example.com/test\n",
		filepath.Join(root, "sample.go"):       "package main\nconst selected = \"root-shadow\"\n",
		filepath.Join(root, "notes.go"):        "package main\n// sample.go root-hit\n",
		filepath.Join(root, "pkg", "notes.go"): "package pkg\n// sample.go scoped-hit\n",
	})

	result, _ := runGatherContext(t, newGatherContextExecCtx(root), map[string]string{
		"query":       "sample.go",
		"path":        "pkg",
		"file_filter": "go",
	})
	assertGatherContextContainsAll(t, result, "Route: Direct query", "Error: direct path not found: sample.go")
	assertGatherContextExcludesAll(t, result, "root-hit", "root-shadow", "scoped-hit", "No matches found", "Route: Auto search")
}
