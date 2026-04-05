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

func TestSearchCode_SnapshotBackedSectionItemAffectedFilesUseProjectRoot(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"pkg/run.go": `package pkg

func Run() {}
`,
		"pkg/run_test.go": `package pkg

func TestRun() {
	Run()
}
`,
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
		ProjectMapStateKey: "snapshot-section-item-subdir",
	}

	result := ExecuteSearchCodeWithCache(cache, opts)
	if !strings.Contains(result, "Related Tests") || !strings.Contains(result, "pkg/run_test.go") {
		t.Fatalf("expected normalized related test path in symbol bundle, got:\n%s", result)
	}

	searchKey := singlePatternBundleCacheKey("Run", cache.lastSetPath)
	affected := cache.affected[searchKey]
	testFile := filepath.Join(dir, "pkg", "run_test.go")
	if !containsAffectedFile(affected, testFile) {
		t.Fatalf("expected section item affected files to include %s, got %v", testFile, affected)
	}
	if containsAffectedFile(affected, filepath.Join(dir, "run_test.go")) {
		t.Fatalf("did not expect wrongly rebased section item path in affected files: %v", affected)
	}

	cache.InvalidateSearchCacheForFile(testFile)
	if _, ok := cache.GetSearch("Run", cache.lastSetPath); ok {
		t.Fatalf("expected symbol cache entry to be invalidated after editing %s", testFile)
	}
}

func TestSearchCode_SnapshotBackedLSPSectionItemKeepsRepoRelativePath(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"pkg/run.go": `package pkg

func Run() {}
`,
		"pkg/run_test.go": `package pkg

func TestRun() {
	Run()
}
`,
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
		ProjectMapStateKey: "snapshot-lsp-section-item-subdir",
		LSPClient: &mockGoSymbolLSPClient{
			refs: []navigation.LSPLocation{
				{File: "pkg/run_test.go", Line: 4, Character: 1, EndLine: 4, EndChar: 5},
			},
		},
	}

	result := ExecuteSearchCodeWithCache(cache, opts)
	if !strings.Contains(result, "Related Tests") || !strings.Contains(result, "pkg/run_test.go:3") {
		t.Fatalf("expected repo-relative LSP section item path, got:\n%s", result)
	}
	if strings.Contains(result, "pkg/pkg/run_test.go") {
		t.Fatalf("did not expect doubly rebased LSP path, got:\n%s", result)
	}

	searchKey := singlePatternBundleCacheKey("Run", cache.lastSetPath)
	affected := cache.affected[searchKey]
	testFile := filepath.Join(dir, "pkg", "run_test.go")
	if !containsAffectedFile(affected, testFile) {
		t.Fatalf("expected LSP section item affected files to include %s, got %v", testFile, affected)
	}

	cache.InvalidateSearchCacheForFile(testFile)
	if _, ok := cache.GetSearch("Run", cache.lastSetPath); ok {
		t.Fatalf("expected symbol cache entry to be invalidated after editing %s", testFile)
	}
}

func TestSearchCode_SnapshotBackedLSPSectionItemPreservesInvocationRelativePath(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"pkg/run.go": `package pkg

func Run() {}
`,
		"shared/run_test.go": `package shared

func TestRunFromShared(t *testing.T) {
	pkg.Run()
}
`,
	})
	subdir := filepath.Join(dir, "pkg")

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
		ProjectMapStateKey: "snapshot-lsp-parent-relative",
		LSPClient: &mockGoSymbolLSPClient{
			refs: []navigation.LSPLocation{
				{File: "../shared/run_test.go", Line: 4, Character: 2, EndLine: 4, EndChar: 6},
			},
		},
	}

	result := ExecuteSearchCodeWithCache(cache, opts)
	if !strings.Contains(result, "Related Tests") || !strings.Contains(result, "shared/run_test.go:3") {
		t.Fatalf("expected parent-relative LSP path to normalize to repo-relative, got:\n%s", result)
	}
	if strings.Contains(result, "../shared/run_test.go") {
		t.Fatalf("did not expect invocation-relative path to leak into output, got:\n%s", result)
	}

	searchKey := singlePatternBundleCacheKey("Run", cache.lastSetPath)
	affected := cache.affected[searchKey]
	testFile := filepath.Join(dir, "shared", "run_test.go")
	if !containsAffectedFile(affected, testFile) {
		t.Fatalf("expected invocation-relative LSP affected files to include %s, got %v", testFile, affected)
	}

	cache.InvalidateSearchCacheForFile(testFile)
	if _, ok := cache.GetSearch("Run", cache.lastSetPath); ok {
		t.Fatalf("expected symbol cache entry to be invalidated after editing %s", testFile)
	}
}

func TestSearchCode_SnapshotBackedLSPSectionItemResolvesBareRelativePathFromInvocationCWD(t *testing.T) {
	testCases := []struct {
		name          string
		lspPath       string
		files         map[string]string
		expectedRel   string
		expectedCache string
		unwantedRel   string
	}{
		{
			name:    "same-dir",
			lspPath: "run_test.go",
			files: map[string]string{
				"pkg/run.go": `package pkg

func Run() {}
`,
				"pkg/run_test.go": `package pkg

func TestRun() {
	Run()
}
`,
			},
			expectedRel:   "pkg/run_test.go:3",
			expectedCache: filepath.Join("pkg", "run_test.go"),
			unwantedRel:   "run_test.go:",
		},
		{
			name:    "child-dir",
			lspPath: "child/run_test.go",
			files: map[string]string{
				"pkg/run.go": `package pkg

func Run() {}
`,
				"pkg/child/run_test.go": `package child

func TestRun() {
	pkg.Run()
}
`,
			},
			expectedRel:   "pkg/child/run_test.go:3",
			expectedCache: filepath.Join("pkg", "child", "run_test.go"),
			unwantedRel:   "child/run_test.go:",
		},
		{
			name:    "invocation-relative-shadow-wins",
			lspPath: "child/run_test.go",
			files: map[string]string{
				"child/run_test.go": `package child

func TestRunFromRoot() {
	pkg.Run()
}
`,
				"pkg/run.go": `package pkg

func Run() {}
`,
				"pkg/child/run_test.go": `package child

func TestRunFromPkgChild() {
	pkg.Run()
}
`,
			},
			expectedRel:   "pkg/child/run_test.go:3",
			expectedCache: filepath.Join("pkg", "child", "run_test.go"),
			unwantedRel:   "child/run_test.go:",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			dir := setupMultiLangDir(t, tc.files)
			subdir := filepath.Join(dir, "pkg")

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
				ProjectMapStateKey: "snapshot-lsp-bare-relative-" + tc.name,
				LSPClient: &mockGoSymbolLSPClient{
					refs: []navigation.LSPLocation{
						{File: tc.lspPath, Line: 4, Character: 2, EndLine: 4, EndChar: 6},
					},
				},
			}

			result := ExecuteSearchCodeWithCache(cache, opts)
			if !strings.Contains(result, "Related Tests") || !strings.Contains(result, tc.expectedRel) {
				t.Fatalf("expected bare relative LSP path to resolve to the correct repo file, got:\n%s", result)
			}
			if tc.unwantedRel != "" && strings.Contains(result, "\n  - "+tc.unwantedRel) {
				t.Fatalf("did not expect unresolved or doubly rebased bare relative path in output, got:\n%s", result)
			}

			searchKey := singlePatternBundleCacheKey("Run", cache.lastSetPath)
			affected := cache.affected[searchKey]
			testFile := filepath.Join(dir, tc.expectedCache)
			if !containsAffectedFile(affected, testFile) {
				t.Fatalf("expected bare relative LSP affected files to include %s, got %v", testFile, affected)
			}

			cache.InvalidateSearchCacheForFile(testFile)
			if _, ok := cache.GetSearch("Run", cache.lastSetPath); ok {
				t.Fatalf("expected symbol cache entry to be invalidated after editing %s", testFile)
			}
		})
	}
}

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
