package search

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/repomap"
)

func TestSearchCode_SymbolFastPathCachesAffectedFiles(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"run.go": "package example\n\nfunc Run() {\n\thelper()\n}\n\nfunc helper() {}\n",
	})

	cache := &testSearchCache{data: make(map[string]string)}
	opts := SearchOptions{
		Pattern:  "Run",
		Path:     dir,
		FileType: "go",
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
		Pattern:  "Run,helper()",
		Path:     dir,
		FileType: "go",
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

	opts := SearchOptions{
		Pattern:       "helper(,Run",
		Path:          dir,
		FileType:      "go",
		InvocationCWD: dir,
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

	normOpts, ok := normalizeSearchOptions(opts)
	if !ok {
		t.Fatal("expected normalized options")
	}
	normOpts.CtxLines = 3
	normOpts.TokenBudget = 15000

	helperOpts := normOpts
	helperOpts.Mode = string(SearchModeLiteral)
	helperExec := executeSinglePatternDetailed(nil, "helper(", helperOpts)
	if !strings.Contains(helperExec.Output, "helper") {
		t.Fatalf("expected helper text search result, got:\n%s", helperExec.Output)
	}
	helperExec.Output = strings.TrimSuffix(helperExec.Output, lineRangeHint)

	runOpts := normOpts
	runOpts.Mode = string(SearchModeSymbol)
	runExec := executeSinglePatternDetailed(nil, "Run", runOpts)
	if !strings.Contains(runExec.Output, "Multiple symbols matched") {
		t.Fatalf("expected symbol-multiple output, got:\n%s", runExec.Output)
	}
	runExec.Output = strings.TrimSuffix(runExec.Output, lineRangeHint)

	affected := collectAffectedFilesFromExecutions([]formattedPatternExecution{
		{Index: 0, singlePatternExecution: helperExec},
		{Index: 1, singlePatternExecution: runExec},
	}, normOpts)

	wantHelper := filepath.Join(dir, "pkg", "helper.go")
	wantLinux := filepath.Join(dir, "pkg", "run_linux.go")
	wantDarwin := filepath.Join(dir, "pkg", "run_darwin.go")
	for _, want := range []string{wantHelper, wantLinux, wantDarwin} {
		if !containsAffectedFile(affected, want) {
			t.Fatalf("expected aggregated affected files to include %s, got %v", want, affected)
		}
	}

	cache := &testSearchCache{data: make(map[string]string)}
	multiKey := buildMultiCacheKey(splitPatterns(normOpts.Pattern))
	cacheKey := "deterministic-multi-cache"
	cache.SetSearch(multiKey, cacheKey, "synthetic multi cache entry", affected)

	cache.InvalidateSearchCacheForFile(wantDarwin)
	if _, ok := cache.GetSearch(multiKey, cacheKey); ok {
		t.Fatalf("expected multi-pattern cache entry to be invalidated after editing %s", wantDarwin)
	}
}

func TestCollectAffectedFilesFromExecutions_SupplementsGoSymbolMultipleWithProjectRoot(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"pkg/helper.go":     "package pkg\n\nfunc helper() {}\n",
		"pkg/run_linux.go":  "package pkg\n\nfunc Run() {}\n",
		"pkg/run_darwin.go": "package pkg\n\nfunc Run() {}\n",
	})

	opts := SearchOptions{
		Pattern:  "helper(,Run",
		Path:     dir,
		FileType: "go",
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
		ProjectMapStateKey: "affected-multi-symbol-multiple-fallback",
	}

	runExec := executeSinglePatternDetailed(nil, "Run", opts)
	if !strings.Contains(runExec.Output, "Multiple symbols matched") {
		t.Fatalf("expected symbol-multiple output, got:\n%s", runExec.Output)
	}
	runExec.AffectedFiles = nil

	wantHelper := filepath.Join(dir, "pkg", "helper.go")
	helperExec := singlePatternExecution{
		Pattern:       "helper(",
		Output:        "📄 pkg/helper.go (1 match)",
		AffectedFiles: []string{wantHelper},
	}

	affected := collectAffectedFilesFromExecutions([]formattedPatternExecution{
		{Index: 0, singlePatternExecution: helperExec},
		{Index: 1, singlePatternExecution: runExec},
	}, opts)

	wantLinux := filepath.Join(dir, "pkg", "run_linux.go")
	wantDarwin := filepath.Join(dir, "pkg", "run_darwin.go")
	for _, want := range []string{wantHelper, wantLinux, wantDarwin} {
		if !containsAffectedFile(affected, want) {
			t.Fatalf("expected supplemented affected files to include %s, got %v", want, affected)
		}
	}
}

func TestCollectAffectedFilesFromExecutions_RepairsWrongGoSymbolMultipleAffectedFiles(t *testing.T) {
	outerDir := t.TempDir()
	dir := filepath.Join(outerDir, "repo")
	if err := os.MkdirAll(filepath.Join(dir, "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	for path, content := range map[string]string{
		"pkg/helper.go":     "package pkg\n\nfunc helper() {}\n",
		"pkg/run_linux.go":  "package pkg\n\nfunc Run() {}\n",
		"pkg/run_darwin.go": "package pkg\n\nfunc Run() {}\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(path)), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	opts := SearchOptions{
		Pattern:  "helper(,Run",
		Path:     dir,
		FileType: "go",
		ProjectMap: &repomap.ProjectMap{
			RootPath: outerDir,
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
		ProjectMapStateKey: "affected-multi-symbol-multiple-wrong-root",
	}

	runExec := executeSinglePatternDetailed(nil, "Run", opts)
	if !strings.Contains(runExec.Output, "Multiple symbols matched") {
		t.Fatalf("expected symbol-multiple output, got:\n%s", runExec.Output)
	}
	runExec.AffectedFiles = []string{
		filepath.Join(outerDir, "pkg", "run_linux.go"),
		filepath.Join(outerDir, "pkg", "run_darwin.go"),
	}

	wantHelper := filepath.Join(dir, "pkg", "helper.go")
	helperExec := singlePatternExecution{
		Pattern:       "helper(",
		Output:        "📄 pkg/helper.go (1 match)",
		AffectedFiles: []string{wantHelper},
	}

	affected := collectAffectedFilesFromExecutions([]formattedPatternExecution{
		{Index: 0, singlePatternExecution: helperExec},
		{Index: 1, singlePatternExecution: runExec},
	}, opts)

	wantLinux := filepath.Join(dir, "pkg", "run_linux.go")
	wantDarwin := filepath.Join(dir, "pkg", "run_darwin.go")
	for _, want := range []string{wantHelper, wantLinux, wantDarwin} {
		if !containsAffectedFile(affected, want) {
			t.Fatalf("expected repaired affected files to include %s, got %v", want, affected)
		}
	}
}

func TestSearchCode_MultiPatternGoSymbolBundleDedupe(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"agent.go": `package example

type Agent struct{}

func (a *Agent) Close() error {
	return nil
}

func run(a *Agent) error {
	return a.Close()
}
`,
		"agent_test.go": `package example

func TestClose() {
	var a Agent
	_ = a.Close()
}
`,
	})

	result := ExecuteSearchCode(SearchOptions{Pattern: `Close,(*Agent).Close,\.Close\(\)`, Path: dir})
	if count := strings.Count(result, "━━ Symbol Bundle:"); count != 1 {
		t.Fatalf("expected a single deduped symbol bundle header, got %d:\n%s", count, result)
	}
	for _, want := range []string{"Matched patterns:", "Close", "(*Agent).Close", `\.Close\(\)`} {
		if !strings.Contains(result, want) {
			t.Errorf("expected %q in deduped bundle output, got:\n%s", want, result)
		}
	}
}

func TestSearchCode_MultiPatternGoSymbolBundleDedupeOnWarmSinglePatternCache(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"agent.go": `package example

type Agent struct{}

func (a *Agent) Close() error {
	return nil
}

func run(a *Agent) error {
	return a.Close()
}
`,
		"agent_test.go": `package example

func TestClose() {
	var a Agent
	_ = a.Close()
}
`,
	})

	cache := &testSearchCache{data: make(map[string]string)}
	opts := SearchOptions{Pattern: `Close,(*Agent).Close,\.Close\(\)`, Path: dir}

	coldResult := ExecuteSearchCodeWithCache(cache, opts)
	if count := strings.Count(coldResult, "━━ Symbol Bundle:"); count != 1 {
		t.Fatalf("expected a single deduped symbol bundle header on cold cache, got %d:\n%s", count, coldResult)
	}

	patterns := splitPatterns(opts.Pattern)
	delete(cache.data, buildMultiCacheKey(patterns)+"|"+buildMultiSearchCacheKey(opts, patterns))

	warmResult := ExecuteSearchCodeWithCache(cache, opts)
	if count := strings.Count(warmResult, "━━ Symbol Bundle:"); count != 1 {
		t.Fatalf("expected a single deduped symbol bundle header on warm single-pattern cache, got %d:\n%s", count, warmResult)
	}
	for _, want := range []string{"Matched patterns:", "Close", "(*Agent).Close", `\.Close\(\)`} {
		if !strings.Contains(warmResult, want) {
			t.Errorf("expected %q in warm-cache deduped bundle output, got:\n%s", want, warmResult)
		}
	}
}

func TestSearchCode_MultiPatternDedupeUnaffectedByUnrelatedInvalidation(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"agent.go": `package example

type Agent struct{}

func (a *Agent) Close() error {
	return nil
}

func run(a *Agent) error {
	return a.Close()
}
`,
		"agent_test.go": `package example

func TestClose() {
	var a Agent
	_ = a.Close()
}
`,
		"unrelated.go": `package example

func noop() {}
`,
	})

	cache := &testSearchCache{data: make(map[string]string)}
	opts := SearchOptions{Pattern: `Close,(*Agent).Close,\.Close\(\)`, Path: dir}

	coldResult := ExecuteSearchCodeWithCache(cache, opts)
	if count := strings.Count(coldResult, "━━ Symbol Bundle:"); count != 1 {
		t.Fatalf("expected a single deduped symbol bundle header on cold cache, got %d:\n%s", count, coldResult)
	}

	patterns := splitPatterns(opts.Pattern)
	delete(cache.data, buildMultiCacheKey(patterns)+"|"+buildMultiSearchCacheKey(opts, patterns))

	cache.InvalidateSearchCacheForFile(filepath.Join(dir, "unrelated.go"))

	warmResult := ExecuteSearchCodeWithCache(cache, opts)
	if count := strings.Count(warmResult, "━━ Symbol Bundle:"); count != 1 {
		t.Fatalf("expected deduped symbol bundle after unrelated invalidation, got %d:\n%s", count, warmResult)
	}
}
