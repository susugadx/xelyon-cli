package search

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/repomap"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

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

func TestCollectAffectedFilesFromExecutions_SupplementsGoSymbolMultipleWithProjectRoot(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"pkg/helper.go":     "package pkg\n\nfunc helper() {}\n",
		"pkg/run_linux.go":  "package pkg\n\nfunc Run() {}\n",
		"pkg/run_darwin.go": "package pkg\n\nfunc Run() {}\n",
	})

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

func TestSinglePatternBundleCacheClearedWithSearchCache(t *testing.T) {
	clearSinglePatternBundleCache()
	t.Cleanup(clearSinglePatternBundleCache)

	dir := setupMultiLangDir(t, map[string]string{
		"agent.go": `package example

type Agent struct{}

func (a *Agent) Close() error {
	return nil
}
`,
	})

	cache := &testSearchCache{data: make(map[string]string)}
	opts := SearchOptions{Pattern: "Close", Path: dir}
	ExecuteSearchCodeWithCache(cache, opts)

	if got := countSinglePatternBundleCacheEntries(); got == 0 {
		t.Fatal("expected bundle cache entry before clear")
	}
	if got := countSinglePatternAffectedFilesCacheEntries(); got == 0 {
		t.Fatal("expected affected-files cache entry before clear")
	}

	cache.ClearSearchCache()

	if got := countSinglePatternBundleCacheEntries(); got != 0 {
		t.Fatalf("expected bundle cache to be cleared, got %d entries", got)
	}
	if got := countSinglePatternAffectedFilesCacheEntries(); got != 0 {
		t.Fatalf("expected affected-files cache to be cleared, got %d entries", got)
	}
}

func TestSinglePatternBundleCacheInvalidatedWithFileInvalidation(t *testing.T) {
	clearSinglePatternBundleCache()
	t.Cleanup(clearSinglePatternBundleCache)

	dir := setupMultiLangDir(t, map[string]string{
		"agent.go": `package example

type Agent struct{}

func (a *Agent) Close() error {
	return nil
}
`,
	})

	cache := &testSearchCache{data: make(map[string]string)}
	opts := SearchOptions{Pattern: "Close", Path: dir}
	normOpts, ok := normalizeSearchOptions(opts)
	if !ok {
		t.Fatal("expected normalized options")
	}
	normOpts.CtxLines = 3
	normOpts.TokenBudget = 15000
	cacheKey := buildSearchCacheKeyWithRoute(normOpts, planSearchRoute("Close", normOpts).cacheSignature())

	storeSinglePatternBundle("Close", cacheKey, &SymbolBundle{
		Identity: SymbolBundleIdentity{Language: "go", Query: "Close", Canonical: "go|agent.go|5|Close", DisplayName: "Close", Kind: "function", File: "agent.go", Line: 5, EndLine: 7},
	})
	storeSinglePatternAffectedFiles("Close", cacheKey, []string{filepath.Join(dir, "agent.go")})
	otherKey := buildSearchCacheKeyWithRoute(normOpts, planSearchRoute("OtherClose", normOpts).cacheSignature())
	storeSinglePatternBundle("OtherClose", otherKey, &SymbolBundle{
		Identity: SymbolBundleIdentity{Language: "go", Query: "OtherClose", Canonical: "go|other.go|5|OtherClose", DisplayName: "OtherClose", Kind: "function", File: "other.go", Line: 5, EndLine: 7},
	})
	storeSinglePatternAffectedFiles("OtherClose", otherKey, []string{filepath.Join(dir, "other.go")})
	cache.SetSearch("Close", cacheKey, "cached", []string{filepath.Join(dir, "agent.go")})
	cache.SetSearch("OtherClose", otherKey, "cached", []string{filepath.Join(dir, "other.go")})

	if got := countSinglePatternBundleCacheEntries(); got != 2 {
		t.Fatalf("expected 2 bundle cache entries before invalidate, got %d", got)
	}
	if got := countSinglePatternAffectedFilesCacheEntries(); got != 2 {
		t.Fatalf("expected 2 affected-files cache entries before invalidate, got %d", got)
	}

	cache.InvalidateSearchCacheForFile(filepath.Join(dir, "agent.go"))

	if got := countSinglePatternBundleCacheEntries(); got != 1 {
		t.Fatalf("expected targeted bundle cache invalidation, got %d entries", got)
	}
	if loadSinglePatternBundle("OtherClose", otherKey) == nil {
		t.Fatal("expected unrelated bundle cache entry to remain")
	}
	if loadSinglePatternAffectedFiles("Close", cacheKey) != nil {
		t.Fatal("expected targeted affected-files cache entry to be removed")
	}
	if loadSinglePatternAffectedFiles("OtherClose", otherKey) == nil {
		t.Fatal("expected unrelated affected-files cache entry to remain")
	}
}

func TestSinglePatternBundleCacheClearedOnSearchCacheEviction(t *testing.T) {
	clearSinglePatternBundleCache()
	t.Cleanup(clearSinglePatternBundleCache)

	storeSinglePatternBundle("keep", "key", &SymbolBundle{Identity: SymbolBundleIdentity{Canonical: "keep"}})
	storeSinglePatternBundle("drop", "key", &SymbolBundle{Identity: SymbolBundleIdentity{Canonical: "drop"}})

	if got := countSinglePatternBundleCacheEntries(); got != 2 {
		t.Fatalf("expected 2 bundle cache entries before eviction, got %d", got)
	}

	tools.NotifySearchCacheEvicted([]string{singlePatternBundleCacheKey("drop", "key")})

	if got := countSinglePatternBundleCacheEntries(); got != 1 {
		t.Fatalf("expected targeted bundle cache eviction, got %d entries", got)
	}
	if loadSinglePatternBundle("keep", "key") == nil {
		t.Fatal("expected unrelated bundle cache entry to remain after eviction")
	}
}

func TestSinglePatternBundleCachePreservesUnrelatedKeysOnTargetedInvalidation(t *testing.T) {
	clearSinglePatternBundleCache()
	t.Cleanup(clearSinglePatternBundleCache)

	storeSinglePatternBundle("keep", "key", &SymbolBundle{Identity: SymbolBundleIdentity{Canonical: "keep"}})
	storeSinglePatternBundle("drop", "key", &SymbolBundle{Identity: SymbolBundleIdentity{Canonical: "drop"}})

	tools.NotifySearchCacheInvalidatedKeys([]string{singlePatternBundleCacheKey("drop", "key")})

	if loadSinglePatternBundle("keep", "key") == nil {
		t.Fatal("expected unrelated bundle cache entry to remain after targeted invalidation")
	}
	if loadSinglePatternBundle("drop", "key") != nil {
		t.Fatal("expected targeted bundle cache entry to be removed")
	}
}
