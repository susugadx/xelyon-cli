package gathercontext

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGatherContext_ExplicitRelativePathDoesNotWidenToProjectRoot(t *testing.T) {
	root := t.TempDir()
	subdir := filepath.Join(root, "pkg")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatal(err)
	}

	writeGatherContextFiles(t, map[string]string{
		filepath.Join(root, "Makefile"): "all:\n",
	})

	result, _ := runGatherContext(t, newGatherContextExecCtx(
		root,
		withGatherContextInvocationCWD(subdir),
	), map[string]string{
		"query":       "./Makefile",
		"file_filter": "go",
	})
	assertGatherContextContainsAll(t, result, "Error: direct path not found: ./Makefile")
	assertGatherContextExcludesAll(t, result, "No matches found", "Route: Auto search")
}

func TestGatherContext_ScopedDirectSurfaceContracts(t *testing.T) {
	root := t.TempDir()
	withGatherContextWorkingDir(t, root)

	writeGatherContextFiles(t, map[string]string{
		filepath.Join(root, "target.go"):                  "package main\nconst selected = \"root\"\n",
		filepath.Join(root, "pkg", "target.go"):           "package pkg\nconst selected = \"subdir\"\n",
		filepath.Join(root, "pkg", "impl.go"):             "package pkg\nconst impl = true\n",
		filepath.Join(root, "pkg", "impl_test.go"):        "package pkg\nconst implTest = true\n",
		filepath.Join(root, "pkg", "nested", "worker.go"): "package nested\nconst selected = \"nested\"\n",
		filepath.Join(root, "pkg", "nested", "README.md"): "nested docs\n",
	})

	execCtx := newGatherContextExecCtx(root)

	t.Run("explicit ranged read ignores scoped search policy", func(t *testing.T) {
		result, _ := runGatherContext(t, execCtx, map[string]string{
			"query":       "target.go:2-2",
			"path":        "pkg",
			"file_filter": "go",
		})
		assertGatherContextContainsAll(t, result, "Route: Direct read", "📄 File: target.go:2-2", `2: const selected = "root"`)
		assertGatherContextExcludesAll(t, result, `"subdir"`)
	})

	t.Run("scoped exact batch uses direct read", func(t *testing.T) {
		result, _ := runGatherContext(t, execCtx, map[string]string{
			"query":       "impl.go,impl_test.go",
			"path":        "pkg",
			"file_filter": "go",
		})
		assertGatherContextContainsAll(t, result, "Route: Direct read", "📄 File: pkg/impl.go", "📄 File: pkg/impl_test.go")
		assertGatherContextExcludesAll(t, result, "No matches found")
	})

	t.Run("explicit directory marker ignores scoped search policy", func(t *testing.T) {
		result, _ := runGatherContext(t, execCtx, map[string]string{
			"query":       "nested/",
			"path":        "pkg",
			"file_filter": "go",
		})
		assertGatherContextContainsAll(t, result, "Route: Direct query", "Error: direct path not found: nested/")
		assertGatherContextExcludesAll(t, result, "Route: Directory listing", "No matches found")
	})
}

func TestGatherContext_ScopedStrictMissesReturnDirectError(t *testing.T) {
	root := t.TempDir()
	withGatherContextWorkingDir(t, root)

	writeGatherContextFiles(t, map[string]string{
		filepath.Join(root, "pkg", "impl.go"): "package pkg\nconst impl = true\n",
	})

	execCtx := newGatherContextExecCtx(root)

	tests := []struct {
		name      string
		query     string
		wantError string
	}{
		{name: "exact", query: "missing.go", wantError: "Error: direct path not found: missing.go"},
		{name: "range", query: "missing.go:1-2", wantError: "Error: direct path not found: missing.go:1-2"},
		{name: "batch", query: "impl.go,missing_test.go", wantError: "Error: direct path not found: missing_test.go"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _ := runGatherContext(t, execCtx, map[string]string{
				"query":       tt.query,
				"path":        "pkg",
				"file_filter": "go",
			})
			assertGatherContextContainsAll(t, result, "Route: Direct query", tt.wantError)
			assertGatherContextExcludesAll(t, result, "No matches found", "Route: Auto search")
		})
	}
}

func TestGatherContext_ScopedAmbiguousExactFilenameReturnsDirectError(t *testing.T) {
	root := t.TempDir()
	withGatherContextWorkingDir(t, root)

	writeGatherContextFiles(t, map[string]string{
		filepath.Join(root, "pkg", "impl.go"):           "package pkg\nconst selected = \"pkg\"\n",
		filepath.Join(root, "pkg", "nested", "impl.go"): "package nested\nconst selected = \"nested\"\n",
	})

	result, _ := runGatherContext(t, newGatherContextExecCtx(root), map[string]string{
		"query":       "impl.go",
		"path":        "pkg",
		"file_filter": "go",
	})
	assertGatherContextContainsAll(t, result, "Route: Direct query", "Error: direct path is ambiguous: impl.go")
	assertGatherContextExcludesAll(t, result, "Route: Auto search", "No matches found")
}

func TestGatherContext_ExplicitDirectoryQueryPreservesFileFilter(t *testing.T) {
	root := t.TempDir()
	withGatherContextWorkingDir(t, root)

	pkgDir := filepath.Join(root, "pkg")
	writeGatherContextFiles(t, map[string]string{
		filepath.Join(pkgDir, "main.go"): "package pkg\n",
		filepath.Join(pkgDir, "main.js"): "export const value = 1;\n",
	})

	execCtx := newGatherContextExecCtx(root)
	for _, query := range []string{"./pkg/", pkgDir + string(os.PathSeparator)} {
		t.Run(strings.ReplaceAll(query, string(os.PathSeparator), "_"), func(t *testing.T) {
			result, _ := runGatherContext(t, execCtx, map[string]string{
				"query":       query,
				"file_filter": "go",
			})
			assertGatherContextContainsAll(t, result, "Route: Directory listing", "main.go")
			assertGatherContextExcludesAll(t, result, "main.js")
		})
	}
}

func TestGatherContext_ExplicitVsSoftDirectContractMatrix(t *testing.T) {
	root := t.TempDir()
	subdir := filepath.Join(root, "pkg", "nested")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatal(err)
	}

	writeGatherContextFiles(t, map[string]string{
		filepath.Join(root, "README.md"):        "# root\n",
		filepath.Join(root, "pkg", "README.md"): "# pkg\n",
	})

	rootExecCtx := newGatherContextExecCtx(root)
	nestedExecCtx := newGatherContextExecCtx(root, withGatherContextInvocationCWD(subdir))

	t.Run("explicit exact file stays direct with matching filter", func(t *testing.T) {
		result, _ := runGatherContext(t, rootExecCtx, map[string]string{
			"query":       "./README.md",
			"file_filter": "md",
		})
		assertGatherContextContainsAll(t, result, "Route: Direct read", "📄 File: README.md", "# root")
		assertGatherContextExcludesAll(t, result, "Route: Auto search", "No matches found")
	})

	t.Run("explicit exact file stays direct with stale filter", func(t *testing.T) {
		result, _ := runGatherContext(t, rootExecCtx, map[string]string{
			"query":       "./README.md",
			"file_filter": "go",
		})
		assertGatherContextContainsAll(t, result, "Route: Direct read", "📄 File: README.md", "# root")
		assertGatherContextExcludesAll(t, result, "Route: Auto search", "No matches found")
	})

	t.Run("soft basename with stale scoped filter falls back to search", func(t *testing.T) {
		result, _ := runGatherContext(t, rootExecCtx, map[string]string{
			"query":       "README.md",
			"path":        "pkg",
			"file_filter": "go",
		})
		assertGatherContextContainsAll(t, result, `Pattern 1/3: "README.md"`)
		assertGatherContextExcludesAll(t, result, "Route: Direct read", "Error: direct path not found", "📄 File:")
	})

	t.Run("soft basename with matching scoped filter resolves scoped direct", func(t *testing.T) {
		result, _ := runGatherContext(t, rootExecCtx, map[string]string{
			"query":       "README.md",
			"path":        "pkg",
			"file_filter": "md",
		})
		assertGatherContextContainsAll(t, result, "Route: Direct read", "📄 File: pkg/README.md", "# pkg")
		assertGatherContextExcludesAll(t, result, "Route: Auto search", "No matches found")
	})

	t.Run("parent relative in repo stays direct", func(t *testing.T) {
		result, _ := runGatherContext(t, nestedExecCtx, map[string]string{
			"query":       "../README.md",
			"file_filter": "go",
		})
		assertGatherContextContainsAll(t, result, "Route: Direct read", "📄 File: ../README.md", "# pkg")
		assertGatherContextExcludesAll(t, result, "Route: Auto search", "No matches found")
	})

	t.Run("escaping parent relative errors", func(t *testing.T) {
		result, _ := runGatherContext(t, nestedExecCtx, map[string]string{
			"query":       "../../outside.txt",
			"file_filter": "go",
		})
		assertGatherContextContainsAll(t, result, "Route: Direct query", "Error: direct path not found: ../../outside.txt")
		assertGatherContextExcludesAll(t, result, "Route: Auto search", "No matches found")
	})
}
