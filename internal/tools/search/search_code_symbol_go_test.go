package search

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/locator"
	"github.com/susugadx/xelyon-cli/internal/navigation"
	"github.com/susugadx/xelyon-cli/internal/repomap"
)

func TestSearchCode_SymbolSingleHit(t *testing.T) {
	setupSymbolTestDir(t, "agent.go", symbolTestSource)

	result := ExecuteSearchCode(SearchOptions{Pattern: "NewAgent", Path: "."})
	if strings.Contains(result, "No matches found") {
		t.Fatal("expected symbol hit, got no matches")
	}
	// symbol auto hit は inspect 形式の結果を返す（lineRangeHint を含まない）
	if strings.Contains(result, lineRangeHint) {
		t.Error("symbol auto result should not contain lineRangeHint")
	}
	if !strings.Contains(result, "NewAgent") {
		t.Error("expected symbol name in result")
	}
}

func TestSearchCode_SymbolMultipleHit(t *testing.T) {
	setupSymbolTestDir(t, "multi.go", symbolTestMultiSource)

	result := ExecuteSearchCode(SearchOptions{Pattern: "Build", Path: "."})
	if !strings.Contains(result, "Multiple symbols matched") {
		t.Errorf("expected multiple symbols result, got: %s", result)
	}
}

func TestSearchCode_SymbolFallbackToText(t *testing.T) {
	setupSymbolTestDir(t, "agent.go", symbolTestSource)

	result := ExecuteSearchCode(SearchOptions{Pattern: "NonExistentXYZ12345", Path: "."})
	if !strings.Contains(result, "No matches found") {
		t.Errorf("expected 'No matches found', got: %s", result)
	}
}

func TestSearchCode_RegexSkipsSymbol(t *testing.T) {
	setupSymbolTestDir(t, "agent.go", symbolTestSource)

	// explicit mode=regex → symbol rescue/resolve をスキップ → text search
	result := ExecuteSearchCode(SearchOptions{Pattern: "NewAgent", Path: ".", Mode: string(SearchModeRegex)})
	// text search 結果は lineRangeHint を含む（symbol auto は IsRegex=true で完全スキップ）
	if !strings.Contains(result, lineRangeHint) {
		t.Error("regex search should fall back to text search with lineRangeHint")
	}
}

func TestSearchCode_UnsupportedLangSkipsSymbol(t *testing.T) {
	setupSymbolTestDir(t, "agent.go", symbolTestSource)

	// 非対応言語 → symbol auto をスキップ → text search
	result := ExecuteSearchCode(SearchOptions{Pattern: "NewAgent", Path: ".", FileType: "haskell"})
	if !strings.Contains(result, "No matches found") && !strings.Contains(result, lineRangeHint) {
		t.Error("unsupported language should fall back to text search")
	}
}

func TestSearchCode_FilePatternSkipsSymbol(t *testing.T) {
	setupSymbolTestDir(t, "agent.go", symbolTestSource)

	// glob パターン指定 → symbol auto をスキップ → text search
	result := ExecuteSearchCode(SearchOptions{Pattern: "NewAgent", Path: ".", FilePattern: "*.go"})
	if !strings.Contains(result, lineRangeHint) {
		t.Error("file pattern should fall back to text search with lineRangeHint")
	}
}

// ── 多言語シンボル解決テスト ──

func TestSearchCode_MultiPatternGoSymbol(t *testing.T) {
	setupSymbolTestDir(t, "agent.go", symbolTestSource)

	// カンマ区切りで Go シンボル2つ → 両方 symbol 解決される
	result := ExecuteSearchCode(SearchOptions{Pattern: "NewAgent,Run", Path: "."})
	if !strings.Contains(result, "Pattern 1/2") {
		t.Error("expected Pattern 1/2 header")
	}
	if !strings.Contains(result, "Pattern 2/2") {
		t.Error("expected Pattern 2/2 header")
	}
	if !strings.Contains(result, "NewAgent") {
		t.Error("expected NewAgent in result")
	}
	if !strings.Contains(result, "Run") {
		t.Error("expected Run in result")
	}
}

func TestGoSymbolResolver_UsesBundleDiagnostics(t *testing.T) {
	setupSymbolTestDir(t, "example.go", `package example

func Run() {}
`)

	resolved := goSymbolResolver{}.Resolve("Run", SearchOptions{
		Path:            ".",
		LSPClient:       &mockGoSymbolLSPClient{refs: []navigation.LSPLocation{{File: "example.go", Line: 3, Character: 1, EndLine: 3, EndChar: 5}}},
		LocatorRegistry: nil,
	})
	if resolved.Status != symbolResolveSingle {
		t.Fatalf("expected single symbol resolution, got %s", resolved.Status)
	}
	if resolved.Bundle == nil {
		t.Fatal("expected bundle in go symbol resolution")
	}
	if !strings.Contains(resolved.Output, "Note: resolved via gopls.") {
		t.Fatalf("expected LSP note in go symbol output, got:\n%s", resolved.Output)
	}
}

func TestGoSymbolResolver_UsesProjectMapSnapshot(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"broken.go": "package example\n\nfunc (\n",
	})

	resolved := goSymbolResolver{}.Resolve("Run", SearchOptions{
		Path: dir,
		ProjectMap: &repomap.ProjectMap{
			RootPath: dir,
			Files: []*repomap.FileEntry{
				{
					Path: "broken.go",
					Symbols: []repomap.Symbol{
						{Name: "Run", Kind: "function", Line: 3, EndLine: 3, Signature: "func Run()", Exported: true},
					},
				},
			},
		},
		ProjectMapRootPath: dir,
		ProjectMapStateKey: "go-snapshot-fast-path",
	})
	if resolved.Status != symbolResolveSingle {
		t.Fatalf("expected snapshot-backed single resolution, got %s", resolved.Status)
	}
	if resolved.Bundle == nil || resolved.Bundle.Definition.File != "broken.go" {
		t.Fatalf("expected snapshot bundle for broken.go, got %+v", resolved.Bundle)
	}
}

func TestGoSymbolResolver_LocatorRegistryDoesNotRegisterHiddenIDs(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"builder.go": `package example

type Builder interface {
	Build() string
}
`,
		"builder_impl.go": `package example

type FileBuilder struct{}

func (FileBuilder) Build() string { return "" }
`,
		"builder_test.go": `package example

func TestBuild(t *testing.T) {
	var b FileBuilder
	_ = b.Build()
}
`,
	})

	reg := locator.NewRegistry()
	resolved := goSymbolResolver{}.Resolve("Builder", SearchOptions{
		Path:            dir,
		LocatorRegistry: reg,
		LSPClient: &mockGoSymbolLSPClient{
			refs:  []navigation.LSPLocation{{File: "builder_test.go", Line: 5, Character: 1, EndLine: 5, EndChar: 6}},
			impls: []navigation.LSPLocation{{File: "builder_impl.go", Line: 3, Character: 1, EndLine: 3, EndChar: 11}},
		},
	})
	if resolved.Status != symbolResolveSingle {
		t.Fatalf("expected single symbol resolution, got %s", resolved.Status)
	}

	ids := visibleLocatorIDs(resolved.Output)
	if len(ids) == 0 {
		t.Fatalf("expected visible locator IDs in output, got:\n%s", resolved.Output)
	}
	for i, id := range ids {
		want := "[L" + strconv.Itoa(i+1) + "]"
		if id != want {
			t.Fatalf("expected sequential locator %s, got %s in output:\n%s", want, id, resolved.Output)
		}
	}
	if _, ok := reg.Resolve("[L" + strconv.Itoa(len(ids)+1) + "]"); ok {
		t.Fatalf("expected no hidden locator beyond visible IDs, got extra registry entry after %d visible IDs", len(ids))
	}
}

func TestGoSymbolResolver_LocatorRegistryMatchesImplementation(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"builder.go": `package example

type Builder interface {
	Build() string
}
`,
		"builder_impl.go": `package example

type FileBuilder struct{}

func (FileBuilder) Build() string { return "" }
`,
	})

	reg := locator.NewRegistry()
	resolved := goSymbolResolver{}.Resolve("Builder", SearchOptions{
		Path:            dir,
		LocatorRegistry: reg,
		LSPClient: &mockGoSymbolLSPClient{
			refs:  []navigation.LSPLocation{{File: "builder_test.go", Line: 5, Character: 1, EndLine: 5, EndChar: 6}},
			impls: []navigation.LSPLocation{{File: "builder_impl.go", Line: 3, Character: 1, EndLine: 3, EndChar: 11}},
		},
	})
	if resolved.Status != symbolResolveSingle {
		t.Fatalf("expected single symbol resolution, got %s", resolved.Status)
	}

	implID := locatorIDForLine(t, resolved.Output, "builder_impl.go:3")
	implLoc, ok := reg.Resolve(implID)
	if !ok {
		t.Fatalf("expected implementation locator %s to resolve", implID)
	}
	if implLoc.FilePath != "builder_impl.go" || implLoc.Line != 3 {
		t.Fatalf("unexpected implementation locator target: %+v", implLoc)
	}
}

func TestFormatSymbolBundle_LocatorRegistryMatchesRelatedTest(t *testing.T) {
	reg := locator.NewRegistry()
	bundle := buildGoSymbolBundle("Close", navigation.InspectResult{
		Symbol: &navigation.SymbolCandidate{
			Name:     "Close",
			Kind:     "method",
			File:     "agent.go",
			Line:     5,
			EndLine:  7,
			Receiver: "*Agent",
		},
		Body: []string{
			"5: func (a *Agent) Close() error {",
			"6: \treturn nil",
			"7: }",
		},
		Tests: []navigation.TestRef{
			{File: "agent_test.go", Line: 4, Name: "TestClose"},
		},
		TotalTests: 1,
	})
	output := formatSymbolBundle(bundle, reg, nil)

	testID := locatorIDForLine(t, output, "agent_test.go:4")
	testLoc, ok := reg.Resolve(testID)
	if !ok {
		t.Fatalf("expected test locator %s to resolve", testID)
	}
	if testLoc.FilePath != "agent_test.go" || testLoc.Line != 4 {
		t.Fatalf("unexpected test locator target: %+v", testLoc)
	}
}

func TestSearchCode_MultiPatternGoSymbolPreservesDiagnostics(t *testing.T) {
	setupSymbolTestDir(t, "example.go", `package example

type Agent struct{}

func (a *Agent) Close() error { return nil }

func run(a *Agent) error {
	return a.Close()
}
`)

	result := ExecuteSearchCode(SearchOptions{
		Pattern: "Close,(*Agent).Close,\\.Close\\(\\)",
		Path:    ".",
		LSPClient: &mockGoSymbolLSPClient{
			refs: []navigation.LSPLocation{{File: "example.go", Line: 5, Character: 1, EndLine: 5, EndChar: 10}},
		},
	})

	if !strings.Contains(result, "Matched patterns:") {
		t.Fatalf("expected multi-pattern bundle output, got:\n%s", result)
	}
	if !strings.Contains(result, "Note: resolved via gopls.") {
		t.Fatalf("expected LSP note in multi-pattern output, got:\n%s", result)
	}
}

func TestSearchCode_SymbolFastPathCachesAffectedFiles(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"run.go": "package example\n\nfunc Run() {\n\thelper()\n}\n\nfunc helper() {}\n",
	})

	cache := &testSearchCache{data: make(map[string]string)}
	opts := SearchOptions{
		Pattern: "Run",
		Path:    dir,
		ProjectMap: &repomap.ProjectMap{
			RootPath: dir,
			Files: []*repomap.FileEntry{
				{
					Path: "run.go",
					Symbols: []repomap.Symbol{
						{Name: "Run", Kind: "function", Line: 3, EndLine: 5, Signature: "func Run()", Exported: true},
					},
				},
			},
		},
		ProjectMapRootPath: dir,
		ProjectMapStateKey: "affected-single",
	}

	result := ExecuteSearchCodeWithCache(cache, opts)
	if !strings.Contains(result, "Run") {
		t.Fatalf("expected symbol result, got:\n%s", result)
	}

	want := filepath.Join(dir, "run.go")
	searchKey := singlePatternBundleCacheKey("Run", cache.lastSetPath)
	affected := cache.affected[searchKey]
	if !containsAffectedFile(affected, want) {
		t.Fatalf("expected exact cache key %q to track %s, got %v", searchKey, want, affected)
	}
}

func TestSearchCode_MultiPatternCacheTracksBundleAndTextAffectedFiles(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"run.go": "package example\n\nfunc Run() {\n\thelper()\n}\n\nfunc helper() {}\n",
	})

	cache := &testSearchCache{data: make(map[string]string)}
	opts := SearchOptions{
		Pattern: "Run,helper()",
		Path:    dir,
		ProjectMap: &repomap.ProjectMap{
			RootPath: dir,
			Files: []*repomap.FileEntry{
				{
					Path: "run.go",
					Symbols: []repomap.Symbol{
						{Name: "Run", Kind: "function", Line: 3, EndLine: 5, Signature: "func Run()", Exported: true},
					},
				},
			},
		},
		ProjectMapRootPath: dir,
		ProjectMapStateKey: "affected-multi",
	}

	result := ExecuteSearchCodeWithCache(cache, opts)
	if !strings.Contains(result, "Run") || !strings.Contains(result, "helper()") {
		t.Fatalf("expected mixed multi-pattern result, got:\n%s", result)
	}

	want := filepath.Join(dir, "run.go")
	searchKey := singlePatternBundleCacheKey(buildMultiCacheKey(splitPatterns(opts.Pattern)), cache.lastSetPath)
	affected := cache.affected[searchKey]
	if !containsAffectedFile(affected, want) {
		t.Fatalf("expected exact multi cache key %q to track %s, got %v", searchKey, want, affected)
	}
}

func TestSearchCode_MultiPatternCacheSupplementsSymbolMultipleAffectedFiles(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"pkg/helper.go":     "package pkg\n\nfunc helper() {}\n",
		"pkg/run_linux.go":  "package pkg\n\nfunc Run() {}\n",
		"pkg/run_darwin.go": "package pkg\n\nfunc Run() {}\n",
	})

	cache := &testSearchCache{data: make(map[string]string)}
	opts := SearchOptions{
		Pattern: "helper(,Run",
		Path:    dir,
		ProjectMap: &repomap.ProjectMap{
			RootPath: dir,
			Files: []*repomap.FileEntry{
				{
					Path: "pkg/run_linux.go",
					Symbols: []repomap.Symbol{
						{Name: "Run", Kind: "function", Line: 3, EndLine: 3, Signature: "func Run()", Exported: true},
					},
				},
				{
					Path: "pkg/run_darwin.go",
					Symbols: []repomap.Symbol{
						{Name: "Run", Kind: "function", Line: 3, EndLine: 3, Signature: "func Run()", Exported: true},
					},
				},
			},
		},
		ProjectMapRootPath: dir,
		ProjectMapStateKey: "affected-multi-symbol-multiple",
	}

	result := ExecuteSearchCodeWithCache(cache, opts)
	if !strings.Contains(result, "Multiple symbols matched") || !strings.Contains(result, "helper") {
		t.Fatalf("expected mixed text/symbol-multiple result, got:\n%s", result)
	}

	searchKey := singlePatternBundleCacheKey(buildMultiCacheKey(splitPatterns(opts.Pattern)), cache.lastSetPath)
	affected := cache.affected[searchKey]
	wantHelper := filepath.Join(dir, "pkg", "helper.go")
	wantLinux := filepath.Join(dir, "pkg", "run_linux.go")
	wantDarwin := filepath.Join(dir, "pkg", "run_darwin.go")
	for _, want := range []string{wantHelper, wantLinux, wantDarwin} {
		if !containsAffectedFile(affected, want) {
			t.Fatalf("expected exact multi cache key %q to track %s, got %v", searchKey, want, affected)
		}
	}

	cache.InvalidateSearchCacheForFile(wantDarwin)
	if _, ok := cache.GetSearch(buildMultiCacheKey(splitPatterns(opts.Pattern)), cache.lastSetPath); ok {
		t.Fatalf("expected multi-pattern cache entry to be invalidated after editing %s", wantDarwin)
	}
}

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

func TestCollectSymbolBundleAffectedFiles_IncludesRecommendedReadFiles(t *testing.T) {
	dir := t.TempDir()
	bundle := &SymbolBundle{
		Definition: SymbolBundleDefinition{
			File: "pkg/run.go",
			Line: 3,
		},
		Impact: &SymbolBundleImpact{
			RiskLevel: "medium",
			RecommendedReads: []SymbolBundleItem{
				{Kind: "references", File: "crosspkg/reader.go", Line: 8, Snippet: "_ = Run()"},
			},
		},
		Debug: SymbolBundleDebug{
			FileRootPath: dir,
		},
	}

	affected := collectSymbolBundleAffectedFiles(bundle, SearchOptions{ProjectMapRootPath: dir})
	wantDefinition := filepath.Join(dir, "pkg", "run.go")
	wantRecommended := filepath.Join(dir, "crosspkg", "reader.go")
	for _, want := range []string{wantDefinition, wantRecommended} {
		if !containsAffectedFile(affected, want) {
			t.Fatalf("expected affected files to include %s, got %v", want, affected)
		}
	}
}

func TestBuildGoSymbolBundleLimitsImplementations(t *testing.T) {
	bundle := buildGoSymbolBundle("Closer", navigation.InspectResult{
		Symbol: &navigation.SymbolCandidate{
			Name:    "Closer",
			Kind:    "interface",
			File:    "closer.go",
			Line:    5,
			EndLine: 7,
		},
		Body: []string{
			"5: type Closer interface {",
			"6: \tClose() error",
			"7: }",
		},
		Implementations: []navigation.ImplementationRef{
			{File: "agent.go", Line: 10, Name: "Agent"},
			{File: "service.go", Line: 20, Name: "Service"},
			{File: "worker.go", Line: 30, Name: "Worker"},
			{File: "job.go", Line: 40, Name: "Job"},
			{File: "task.go", Line: 50, Name: "Task"},
		},
	})

	var implSection *SymbolBundleSection
	for i := range bundle.Sections {
		if bundle.Sections[i].Kind == "implementations" {
			implSection = &bundle.Sections[i]
			break
		}
	}
	if implSection == nil {
		t.Fatal("expected implementations section")
	}
	if len(implSection.Items) != goImplementationLimit {
		t.Fatalf("expected %d implementation items, got %d", goImplementationLimit, len(implSection.Items))
	}
	if implSection.Total != 5 {
		t.Fatalf("expected Total=5, got %d", implSection.Total)
	}
	if !implSection.More {
		t.Fatal("expected More=true when implementations are truncated")
	}
}

func TestBuildGoSymbolBundleKeepsAllImplementationsWhenUnderLimit(t *testing.T) {
	bundle := buildGoSymbolBundle("Closer", navigation.InspectResult{
		Symbol: &navigation.SymbolCandidate{
			Name:    "Closer",
			Kind:    "interface",
			File:    "closer.go",
			Line:    5,
			EndLine: 7,
		},
		Body: []string{
			"5: type Closer interface {",
			"6: \tClose() error",
			"7: }",
		},
		Implementations: []navigation.ImplementationRef{
			{File: "agent.go", Line: 10, Name: "Agent"},
			{File: "service.go", Line: 20, Name: "Service"},
		},
	})

	var implSection *SymbolBundleSection
	for i := range bundle.Sections {
		if bundle.Sections[i].Kind == "implementations" {
			implSection = &bundle.Sections[i]
			break
		}
	}
	if implSection == nil {
		t.Fatal("expected implementations section")
	}
	if len(implSection.Items) != 2 {
		t.Fatalf("expected 2 implementation items, got %d", len(implSection.Items))
	}
	if implSection.Total != 2 {
		t.Fatalf("expected Total=2, got %d", implSection.Total)
	}
	if implSection.More {
		t.Fatal("expected More=false when implementations are not truncated")
	}
}
