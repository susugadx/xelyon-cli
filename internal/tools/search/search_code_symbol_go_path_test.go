package search

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/navigation"
	"github.com/susugadx/xelyon-cli/internal/repomap"
)

func TestSearchCode_SymbolBundleAffectedFilesStayRepoRelativeFromSubdir(t *testing.T) {
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
					Path: "pkg/run.go",
					Symbols: []repomap.Symbol{
						{Name: "Run", Kind: "function", Line: 3, EndLine: 3, Signature: "func Run()", Exported: true},
					},
				},
			},
		},
		ProjectMapStateKey: "symbol-subdir-root",
	}

	result := ExecuteSearchCodeWithCache(cache, opts)
	if !strings.Contains(result, "in pkg/run.go") {
		t.Fatalf("expected repo-relative symbol bundle path, got:\n%s", result)
	}

	searchKey := singlePatternBundleCacheKey("Run", cache.lastSetPath)
	affected := cache.affected[searchKey]
	want := filepath.Join(dir, "pkg", "run.go")
	if !containsAffectedFile(affected, want) {
		t.Fatalf("expected symbol bundle affected files to include %s, got %v", want, affected)
	}
	if containsAffectedFile(affected, filepath.Join(dir, "run.go")) {
		t.Fatalf("did not expect wrongly rebased root path in affected files: %v", affected)
	}
}
func TestGoSymbolScopesUseInvocationCWDForRelativeDirectoryPath(t *testing.T) {
	root := t.TempDir()
	cwd := filepath.Join(root, "workspace")
	if err := os.MkdirAll(filepath.Join(cwd, "app", "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	opts := SearchOptions{
		Path:               "app",
		ProjectMapRootPath: root,
		InvocationCWD:      cwd,
	}
	want := filepath.Join(cwd, "app")

	scope := goSymbolSearchScopeForOptions(opts)
	if scope.DefinitionPathHint != want {
		t.Fatalf("DefinitionPathHint = %q, want %q", scope.DefinitionPathHint, want)
	}
	if scope.FallbackReferenceSearchPath != want {
		t.Fatalf("FallbackReferenceSearchPath = %q, want %q", scope.FallbackReferenceSearchPath, want)
	}
	if scope.ReferenceFilter == nil {
		t.Fatal("ReferenceFilter should be set for directory scope")
	}
	if !scope.ReferenceFilter(navigation.Reference{ResolvedPath: filepath.Join(want, "use.go")}) {
		t.Fatal("ReferenceFilter rejected same-directory file")
	}
	if !scope.ReferenceFilter(navigation.Reference{ResolvedPath: filepath.Join(want, "sub", "use.go")}) {
		t.Fatal("ReferenceFilter rejected directory subtree file")
	}
	if scope.ReferenceFilter(navigation.Reference{ResolvedPath: filepath.Join(cwd, "other", "use.go")}) {
		t.Fatal("ReferenceFilter accepted sibling directory file")
	}
}
func TestGoSymbolScopesKeepDirectFileReferencesInPackageDir(t *testing.T) {
	root := t.TempDir()
	filePath := filepath.Join(root, "pkg", "run.go")
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filePath, []byte("package pkg\n\nfunc Run() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	opts := SearchOptions{
		Path:               filePath,
		ProjectMapRootPath: root,
		InvocationCWD:      root,
	}
	scope := goSymbolSearchScopeForOptions(opts)
	if scope.DefinitionPathHint != filePath {
		t.Fatalf("DefinitionPathHint = %q, want direct file %q", scope.DefinitionPathHint, filePath)
	}
	packageDir := filepath.Join(root, "pkg")
	if scope.FallbackReferenceSearchPath != packageDir {
		t.Fatalf("FallbackReferenceSearchPath = %q, want package dir %q", scope.FallbackReferenceSearchPath, packageDir)
	}
	if scope.ReferenceFilter == nil {
		t.Fatal("ReferenceFilter should be set for direct-file package scope")
	}
	if !scope.ReferenceFilter(navigation.Reference{ResolvedPath: filepath.Join(packageDir, "use.go")}) {
		t.Fatal("ReferenceFilter rejected same-package sibling file")
	}
	if scope.ReferenceFilter(navigation.Reference{ResolvedPath: filepath.Join(packageDir, "sub", "use.go")}) {
		t.Fatal("ReferenceFilter accepted subpackage file")
	}
	if scope.ReferenceFilter(navigation.Reference{ResolvedPath: filepath.Join(root, "other", "use.go")}) {
		t.Fatal("ReferenceFilter accepted other package file")
	}
}
func TestCollectNavigationCandidatesAffectedFiles_PrefersExistingProjectRoot(t *testing.T) {
	outerDir := t.TempDir()
	dir := filepath.Join(outerDir, "repo")
	if err := os.MkdirAll(filepath.Join(dir, "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"run_linux.go", "run_darwin.go"} {
		if err := os.WriteFile(filepath.Join(dir, "pkg", name), []byte("package pkg\n\nfunc Run() {}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	affected := collectNavigationCandidatesAffectedFiles([]navigation.SymbolCandidate{
		{File: "pkg/run_linux.go", RootPath: outerDir},
		{File: "pkg/run_darwin.go", RootPath: outerDir},
	}, SearchOptions{
		Path:               dir,
		ProjectMapRootPath: dir,
	})

	wantLinux := filepath.Join(dir, "pkg", "run_linux.go")
	wantDarwin := filepath.Join(dir, "pkg", "run_darwin.go")
	for _, want := range []string{wantLinux, wantDarwin} {
		if !containsAffectedFile(affected, want) {
			t.Fatalf("expected affected files to include %s, got %v", want, affected)
		}
	}
	if containsAffectedFile(affected, filepath.Join(outerDir, "pkg", "run_linux.go")) {
		t.Fatalf("did not expect broader root path to win: %v", affected)
	}
}
