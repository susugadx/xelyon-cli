package search

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type recentActivitySearchCache struct {
	*testSearchCache
	recentFilePaths           []string
	recentSearchAffectedFiles []string
	recentSearchExcluding     []string
	excludingPattern          string
	excludingCacheKey         string
}

type recentActivityOnlySearchCache struct {
	*testSearchCache
	recentFilePaths           []string
	recentSearchAffectedFiles []string
}

func (c *recentActivitySearchCache) RecentFilePaths(limit int) []string {
	return limitPaths(c.recentFilePaths, limit)
}

func (c *recentActivitySearchCache) RecentSearchAffectedFiles(limit int) []string {
	return limitPaths(c.recentSearchAffectedFiles, limit)
}

func (c *recentActivitySearchCache) RecentSearchAffectedFilesExcluding(pattern, cacheKey string, limit int) []string {
	c.excludingPattern = pattern
	c.excludingCacheKey = cacheKey
	if c.recentSearchExcluding != nil {
		return limitPaths(c.recentSearchExcluding, limit)
	}
	return limitPaths(c.recentSearchAffectedFiles, limit)
}

func (c *recentActivityOnlySearchCache) RecentFilePaths(limit int) []string {
	return limitPaths(c.recentFilePaths, limit)
}

func (c *recentActivityOnlySearchCache) RecentSearchAffectedFiles(limit int) []string {
	return limitPaths(c.recentSearchAffectedFiles, limit)
}

func limitPaths(paths []string, limit int) []string {
	if limit <= 0 || len(paths) == 0 {
		return nil
	}
	if len(paths) > limit {
		paths = paths[:limit]
	}
	return append([]string(nil), paths...)
}

func TestRankImpactBundleForRuntime_NoRecentActivityKeepsOrder(t *testing.T) {
	root, bundle, paths := setupImpactRankingFixture(t)

	cache := &recentActivitySearchCache{
		testSearchCache: &testSearchCache{data: make(map[string]string)},
	}

	ranked := rankImpactBundleForRuntime(bundle, SearchOptions{
		ProjectMapRootPath: root,
		InvocationCWD:      filepath.Join(root, "pkg"),
	}, cache)

	want := []string{
		"pkg/run.go",
		"pkg/caller.go",
		"shadow.go",
		"pkg/run_test.go",
		"pkg/other.go",
	}
	if got := recommendedReadFiles(ranked); !equalStrings(got, want) {
		t.Fatalf("expected recommended reads order %v, got %v", want, got)
	}
	if got := recommendedReadFiles(bundle); !equalStrings(got, want) {
		t.Fatalf("expected original bundle order to stay unchanged, got %v", got)
	}
	if paths["shadowResolved"] == "" {
		t.Fatal("expected fixture to provide resolved shadow path")
	}
}

func TestRankImpactBundleForRuntime_RecentFilePathsBoostsResolvedPathMatch(t *testing.T) {
	root, bundle, paths := setupImpactRankingFixture(t)

	cache := &recentActivitySearchCache{
		testSearchCache: &testSearchCache{data: make(map[string]string)},
		recentFilePaths: []string{paths["shadowResolved"]},
	}

	ranked := rankImpactBundleForRuntime(bundle, SearchOptions{
		ProjectMapRootPath: root,
		InvocationCWD:      filepath.Join(root, "pkg"),
	}, cache)

	want := []string{
		"pkg/run.go",
		"shadow.go",
		"pkg/caller.go",
		"pkg/run_test.go",
		"pkg/other.go",
	}
	if got := recommendedReadFiles(ranked); !equalStrings(got, want) {
		t.Fatalf("expected recent file path boost order %v, got %v", want, got)
	}
}

func TestRankImpactBundleForRuntime_RecentSearchAffectedFilesBoostsMatch(t *testing.T) {
	root, bundle, paths := setupImpactRankingFixture(t)

	cache := &recentActivitySearchCache{
		testSearchCache:           &testSearchCache{data: make(map[string]string)},
		recentSearchAffectedFiles: []string{paths["test"]},
	}

	ranked := rankImpactBundleForRuntime(bundle, SearchOptions{
		ProjectMapRootPath: root,
		InvocationCWD:      filepath.Join(root, "pkg"),
	}, cache)

	want := []string{
		"pkg/run.go",
		"pkg/run_test.go",
		"pkg/caller.go",
		"shadow.go",
		"pkg/other.go",
	}
	if got := recommendedReadFiles(ranked); !equalStrings(got, want) {
		t.Fatalf("expected recent affected-file boost order %v, got %v", want, got)
	}
}

func TestRankImpactBundleForRuntime_StableTieKeepsOriginalOrder(t *testing.T) {
	root, bundle, paths := setupImpactRankingFixture(t)

	cache := &recentActivitySearchCache{
		testSearchCache:           &testSearchCache{data: make(map[string]string)},
		recentSearchAffectedFiles: []string{paths["caller"], paths["test"]},
	}

	ranked := rankImpactBundleForRuntime(bundle, SearchOptions{
		ProjectMapRootPath: root,
		InvocationCWD:      filepath.Join(root, "pkg"),
	}, cache)

	want := []string{
		"pkg/run.go",
		"pkg/caller.go",
		"pkg/run_test.go",
		"shadow.go",
		"pkg/other.go",
	}
	if got := recommendedReadFiles(ranked); !equalStrings(got, want) {
		t.Fatalf("expected stable tie order %v, got %v", want, got)
	}
}

func TestRankImpactBundleForRuntime_NoOptionalInterfaceKeepsOrder(t *testing.T) {
	root, bundle, _ := setupImpactRankingFixture(t)

	ranked := rankImpactBundleForRuntime(bundle, SearchOptions{
		ProjectMapRootPath: root,
		InvocationCWD:      filepath.Join(root, "pkg"),
	}, &testSearchCache{data: make(map[string]string)})

	want := []string{
		"pkg/run.go",
		"pkg/caller.go",
		"shadow.go",
		"pkg/run_test.go",
		"pkg/other.go",
	}
	if got := recommendedReadFiles(ranked); !equalStrings(got, want) {
		t.Fatalf("expected order to stay unchanged without optional cache interface, got %v", got)
	}
}

func TestRankImpactBundleForRuntime_ExcludeCurrentSearchWithoutExactProviderKeepsOrder(t *testing.T) {
	root, bundle, paths := setupImpactRankingFixture(t)
	cache := &recentActivityOnlySearchCache{
		testSearchCache:           &testSearchCache{data: make(map[string]string)},
		recentSearchAffectedFiles: []string{paths["other"]},
	}

	ranked := rankImpactBundleForRuntimeWithContext(bundle, SearchOptions{
		ProjectMapRootPath: root,
		InvocationCWD:      filepath.Join(root, "pkg"),
	}, cache, currentSearchImpactRuntimeRankContext("Run", "cache-key"))

	want := []string{
		"pkg/run.go",
		"pkg/caller.go",
		"shadow.go",
		"pkg/run_test.go",
		"pkg/other.go",
	}
	if got := recommendedReadFiles(ranked); !equalStrings(got, want) {
		t.Fatalf("expected cache-hit ranking without exact exclusion support to stay unchanged, got %v", got)
	}
}

func TestExecuteSinglePatternDetailed_CacheHitReRanksImpactBundle(t *testing.T) {
	clearSinglePatternBundleCache()
	t.Cleanup(clearSinglePatternBundleCache)

	root, bundle, paths := setupImpactRankingFixture(t)
	opts := normalizedImpactSearchOptionsForTest(t, root)
	cache := &recentActivitySearchCache{
		testSearchCache: &testSearchCache{data: make(map[string]string)},
		recentFilePaths: []string{paths["other"]},
	}

	cacheKey := buildSearchCacheKeyWithRoute(opts, planSearchRoute("Run", opts).cacheSignature())
	storeSinglePatternBundle("Run", cacheKey, bundle)
	cache.SetSearch("Run", cacheKey, "stale cached output", collectSymbolBundleAffectedFiles(bundle, opts))

	first := executeSinglePatternDetailed(cache, "Run", opts)
	if !strings.Contains(first.Output, "pkg/other.go:30") {
		t.Fatalf("expected reranked cached output to include pkg/other.go near the front, got:\n%s", first.Output)
	}
	if strings.Contains(first.Output, "stale cached output") {
		t.Fatalf("expected cached string fallback to be bypassed when bundle is available, got:\n%s", first.Output)
	}
	if got := first.Bundle.Impact.RecommendedReads[1].File; got != "pkg/other.go" {
		t.Fatalf("expected runtime-ranked bundle to surface pkg/other.go first after definition, got %s", got)
	}

	cache.recentFilePaths = []string{paths["caller"]}
	second := executeSinglePatternDetailed(cache, "Run", opts)
	if got := second.Bundle.Impact.RecommendedReads[1].File; got != "pkg/caller.go" {
		t.Fatalf("expected cache hit reranking to reflect updated recent activity, got %s", got)
	}
	if cache.setCalls != 1 {
		t.Fatalf("expected cache hit reranking to reuse the same cache entry without rewriting it, setCalls=%d", cache.setCalls)
	}
}

func TestTryStructuredGoImpactSearch_CacheHitReRanksRecommendedReads(t *testing.T) {
	clearSinglePatternBundleCache()
	t.Cleanup(clearSinglePatternBundleCache)

	root, bundle, paths := setupImpactRankingFixture(t)
	opts := normalizedImpactSearchOptionsForTest(t, root)
	cache := &recentActivitySearchCache{
		testSearchCache: &testSearchCache{data: make(map[string]string)},
		recentFilePaths: []string{paths["other"]},
	}

	cacheKey := buildSearchCacheKeyWithRoute(opts, planSearchRoute("Run", opts).cacheSignature()+"|"+structuredGoImpactRouteTag)
	storeSinglePatternBundle("Run", cacheKey, bundle)
	cache.SetSearch("Run", cacheKey, "stale cached output", collectSymbolBundleAffectedFiles(bundle, opts))

	first, ok := tryStructuredGoImpactSearch(cache, opts)
	if !ok {
		t.Fatal("expected structured impact cache hit")
	}
	assertRecommendedReadOrder(t, first, bundle.Impact.RecommendedReads[0], bundle.Impact.RecommendedReads[4], bundle.Impact.RecommendedReads[1])

	cache.recentFilePaths = []string{paths["caller"]}
	second, ok := tryStructuredGoImpactSearch(cache, opts)
	if !ok {
		t.Fatal("expected structured impact cache hit on second lookup")
	}
	assertRecommendedReadOrder(t, second, bundle.Impact.RecommendedReads[0], bundle.Impact.RecommendedReads[1], bundle.Impact.RecommendedReads[4])

	if cache.setCalls != 1 {
		t.Fatalf("expected structured impact cache hit reranking to reuse the same cache entry, setCalls=%d", cache.setCalls)
	}
}

func TestExecuteSinglePatternDetailed_CacheHitRecentSearchExcludesCurrentEntry(t *testing.T) {
	clearSinglePatternBundleCache()
	t.Cleanup(clearSinglePatternBundleCache)

	root, bundle, paths := setupImpactRankingFixture(t)
	opts := normalizedImpactSearchOptionsForTest(t, root)
	cache := &recentActivitySearchCache{
		testSearchCache:           &testSearchCache{data: make(map[string]string)},
		recentSearchAffectedFiles: []string{paths["caller"], paths["test"], paths["other"], paths["shadowResolved"]},
		recentSearchExcluding:     []string{paths["other"]},
	}

	cacheKey := buildSearchCacheKeyWithRoute(opts, planSearchRoute("Run", opts).cacheSignature())
	storeSinglePatternBundle("Run", cacheKey, bundle)
	currentAffected := collectSymbolBundleAffectedFiles(bundle, opts)
	cache.SetSearch("Run", cacheKey, "stale cached output", currentAffected)

	result := executeSinglePatternDetailed(cache, "Run", opts)
	if got := result.Bundle.Impact.RecommendedReads[1].File; got != "pkg/other.go" {
		t.Fatalf("expected recent search reranking to exclude current cache entry and surface pkg/other.go, got %s", got)
	}
	if cache.excludingPattern != "Run" || cache.excludingCacheKey != cacheKey {
		t.Fatalf("expected exclusion provider to receive current search identity, got pattern=%q cacheKey=%q", cache.excludingPattern, cache.excludingCacheKey)
	}
}

func setupImpactRankingFixture(t *testing.T) (string, *SymbolBundle, map[string]string) {
	t.Helper()

	root := t.TempDir()
	files := map[string]string{
		"pkg/run.go":      "package pkg\n\nfunc Run() error { return nil }\n",
		"pkg/caller.go":   "package pkg\n\nfunc callRun() { _ = Run() }\n",
		"pkg/run_test.go": "package pkg\n\nfunc TestRun(t *testing.T) { _ = Run() }\n",
		"pkg/other.go":    "package pkg\n\nfunc other() { _ = Run() }\n",
		"shadow.go":       "package main\n\nfunc rootShadow() {}\n",
		"sub/shadow.go":   "package sub\n\nfunc resolvedShadow() {}\n",
	}
	for relPath, content := range files {
		absPath := filepath.Join(root, filepath.FromSlash(relPath))
		if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	paths := map[string]string{
		"caller":         filepath.Join(root, "pkg", "caller.go"),
		"test":           filepath.Join(root, "pkg", "run_test.go"),
		"other":          filepath.Join(root, "pkg", "other.go"),
		"shadowResolved": filepath.Join(root, "sub", "shadow.go"),
	}

	bundle := &SymbolBundle{
		Identity: SymbolBundleIdentity{
			Language:    "go",
			Query:       "Run",
			Canonical:   "go|pkg/run.go|3|Run",
			DisplayName: "Run",
			Kind:        "function",
			File:        "pkg/run.go",
			Line:        3,
			EndLine:     3,
		},
		Definition: SymbolBundleDefinition{
			File:      "pkg/run.go",
			Line:      3,
			EndLine:   3,
			Signature: "func Run() error",
			Body: []string{
				"3: func Run() error {",
				"4: \treturn nil",
				"5: }",
			},
		},
		Impact: &SymbolBundleImpact{
			RiskLevel: "medium",
			RecommendedReads: []SymbolBundleItem{
				{Kind: "definition", File: "pkg/run.go", Line: 3, EndLine: 3, Snippet: "func Run() error", Name: "Run"},
				{Kind: "callers", File: "pkg/caller.go", Line: 10, Snippet: "_ = Run()"},
				{Kind: "references", File: "shadow.go", ResolvedPath: paths["shadowResolved"], Line: 20, Snippet: "_ = Run()"},
				{Kind: "tests", File: "pkg/run_test.go", Line: 5, Name: "TestRun"},
				{Kind: "references", File: "pkg/other.go", Line: 30, Snippet: "_ = Run()"},
			},
		},
		Debug: SymbolBundleDebug{
			FileRootPath: root,
		},
	}

	return root, bundle, paths
}

func normalizedImpactSearchOptionsForTest(t *testing.T, root string) SearchOptions {
	t.Helper()

	opts, ok := normalizeSearchOptions(SearchOptions{
		Pattern:        "Run",
		Intent:         "impact",
		Path:           root,
		FileType:       "go",
		IncludeIgnored: true,
	})
	if !ok {
		t.Fatal("expected normalized search options")
	}
	opts.CtxLines = 3
	opts.TokenBudget = 15000
	opts.InvocationCWD = root
	opts.ProjectMapRootPath = root
	return opts
}

func equalStrings(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func assertRecommendedReadOrder(t *testing.T, output string, items ...SymbolBundleItem) {
	t.Helper()

	lastIdx := -1
	for _, item := range items {
		fragment := "  - " + formatSymbolBundleItem(item.Kind, item)
		idx := strings.Index(output, fragment)
		if idx < 0 {
			t.Fatalf("expected output to contain recommended read line %q, got:\n%s", fragment, output)
		}
		if idx <= lastIdx {
			t.Fatalf("expected recommended read line %q to appear after previous line, got:\n%s", fragment, output)
		}
		lastIdx = idx
	}
}
