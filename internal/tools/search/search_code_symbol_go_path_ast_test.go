package search

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/repomap"
)

func TestSearchCode_SymbolBundleAffectedFilesUseInvocationCWDOnASTFallback(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"pkg/run.go": "package pkg\n\nfunc Run() {}\n",
	})
	subdir := filepath.Join(dir, "pkg")

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(subdir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origDir)
	})

	cache := &testSearchCache{data: make(map[string]string)}
	opts := SearchOptions{
		Pattern:            "Run",
		Path:               ".",
		ProjectMapRootPath: dir,
		InvocationCWD:      subdir,
		ProjectMap: &repomap.ProjectMap{
			RootPath: dir,
			Files: []*repomap.FileEntry{
				{
					Path:    "pkg/other.go",
					Symbols: []repomap.Symbol{{Name: "Other", Kind: "function", Line: 3, EndLine: 3, Signature: "func Other()", Exported: true}},
				},
			},
		},
		ProjectMapStateKey: "symbol-ast-fallback-subdir",
	}

	result := ExecuteSearchCodeWithCache(cache, opts)
	if !strings.Contains(result, "func Run()") {
		t.Fatalf("expected AST fallback symbol result, got:\n%s", result)
	}

	searchKey := singlePatternBundleCacheKey("Run", cache.lastSetPath)
	affected := cache.affected[searchKey]
	want := filepath.Join(dir, "pkg", "run.go")
	if !containsAffectedFile(affected, want) {
		t.Fatalf("expected AST fallback affected files to include %s, got %v", want, affected)
	}
	if containsAffectedFile(affected, filepath.Join(dir, "run.go")) {
		t.Fatalf("did not expect wrongly rebased repo-root path in affected files: %v", affected)
	}

	cache.InvalidateSearchCacheForFile(want)
	if _, ok := cache.GetSearch("Run", cache.lastSetPath); ok {
		t.Fatalf("expected symbol cache entry to be invalidated after editing %s", want)
	}
}
