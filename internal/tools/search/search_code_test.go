package search

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/locator"
)

func TestSearchCode_CacheKeyUsesInternalTokenBudget(t *testing.T) {
	setupSearchTestMocks(t)

	dir := t.TempDir()
	file1 := filepath.Join(dir, "cached.go")
	if err := os.WriteFile(file1, []byte("func cached_target() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	cache := &testSearchCache{data: make(map[string]string)}

	result1 := ExecuteSearchCodeWithCache(cache, SearchOptions{Pattern: "cached_target", Path: dir, CtxLines: 0, TokenBudget: 500, IsRegex: true})
	if strings.Contains(result1, "No matches found") {
		t.Fatal("Expected matches on first search")
	}
	if !strings.Contains(cache.lastSetPath, "|3|15000|") {
		t.Fatalf("expected cache key to use internal defaults for context_lines=3 and token_budget=15000, got: %s", cache.lastSetPath)
	}

	result2 := ExecuteSearchCodeWithCache(cache, SearchOptions{Pattern: "cached_target", Path: dir, CtxLines: 0, TokenBudget: 99999, IsRegex: true})
	if result2 != result1 {
		t.Fatal("Expected second result to be served from the same cache key")
	}
	if cache.setCalls != 1 {
		t.Fatalf("expected one cache write with normalized token budget, got %d", cache.setCalls)
	}
	if !strings.Contains(cache.lastGetPath, "|3|15000|") {
		t.Fatalf("expected cache lookup key to use internal defaults for context_lines=3 and token_budget=15000, got: %s", cache.lastGetPath)
	}
}

func TestSearchCode_AffectedFilesUseInvocationCWDForSubdirSearch(t *testing.T) {
	setupSearchTestMocks(t)

	root := t.TempDir()
	subdir := filepath.Join(root, "pkg")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatal(err)
	}
	targetFile := filepath.Join(subdir, "target.go")
	if err := os.WriteFile(targetFile, []byte("package pkg\n\nconst target_text = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

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
		Pattern:            "target_text",
		Path:               ".",
		Mode:               string(SearchModeLiteral),
		ProjectMapRootPath: root,
		InvocationCWD:      subdir,
	}

	result := ExecuteSearchCodeWithCache(cache, opts)
	if !strings.Contains(result, "target.go") {
		t.Fatalf("expected text search result, got:\n%s", result)
	}

	searchKey := singlePatternBundleCacheKey("target_text", cache.lastSetPath)
	affected := cache.affected[searchKey]
	if len(affected) != 1 || affected[0] != targetFile {
		t.Fatalf("expected affected files [%s], got %v", targetFile, affected)
	}

	cache.InvalidateSearchCacheForFile(targetFile)
	if _, ok := cache.GetSearch("target_text", cache.lastSetPath); ok {
		t.Fatal("expected text search cache entry to be invalidated after file edit")
	}
}

func TestSearchCode_SubdirTextSearchLocatorsCarryResolvedPath(t *testing.T) {
	setupSearchTestMocks(t)

	root := t.TempDir()
	subdir := filepath.Join(root, "pkg")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatal(err)
	}
	rootTarget := filepath.Join(root, "target.go")
	subdirTarget := filepath.Join(subdir, "target.go")
	if err := os.WriteFile(rootTarget, []byte("package main\n\nconst root_only = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(subdirTarget, []byte("package pkg\n\nconst target_text = 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

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

	reg := locator.NewRegistry()
	result := ExecuteSearchCode(SearchOptions{
		Pattern:            "target_text",
		Path:               ".",
		Mode:               string(SearchModeLiteral),
		LocatorRegistry:    reg,
		ProjectMapRootPath: root,
		InvocationCWD:      subdir,
	})
	if !strings.Contains(result, "target.go") {
		t.Fatalf("expected text search result, got:\n%s", result)
	}

	loc, ok := reg.Resolve("[L1]")
	if !ok {
		t.Fatal("expected file locator [L1] to be registered")
	}
	if filepath.Clean(loc.FilePath) != "target.go" {
		t.Fatalf("expected display path target.go, got %+v", loc)
	}
	if loc.ResolvedPath != subdirTarget {
		t.Fatalf("expected resolved path %s, got %+v", subdirTarget, loc)
	}
}

func TestExtractPrimaryFileRefs_GenericMultipleDefsUseInvocationCWD(t *testing.T) {
	root := t.TempDir()
	subdir := filepath.Join(root, "pkg")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatal(err)
	}

	rootTarget := filepath.Join(root, "target.py")
	subdirTarget := filepath.Join(subdir, "target.py")
	if err := os.WriteFile(rootTarget, []byte("def root_only():\n    pass\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(subdirTarget, []byte("def Foo():\n    return 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	output := strings.Join([]string{
		`Multiple definitions found for "Foo":`,
		`  1. function Foo (L1) in target.py [L1]`,
		"",
		`Refine with path to disambiguate (e.g. path="src/models/").`,
	}, "\n")

	refs := extractPrimaryFileRefs(output, SearchOptions{
		ProjectMapRootPath: root,
		InvocationCWD:      subdir,
	})
	if len(refs) != 1 {
		t.Fatalf("expected one extracted ref, got %v", refs)
	}
	if refs[0].DisplayPath != "target.py" {
		t.Fatalf("expected display path target.py, got %+v", refs[0])
	}
	if refs[0].ResolvedPath != subdirTarget {
		t.Fatalf("expected invocation-relative resolved path %s, got %+v", subdirTarget, refs[0])
	}
}

func TestBuildCrossPatternIndexFromExecutions_BundleUsesExecutionMetadataNotCurrentRegistry(t *testing.T) {
	root := t.TempDir()
	subdir := filepath.Join(root, "pkg")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatal(err)
	}

	rootTarget := filepath.Join(root, "target.py")
	subdirTarget := filepath.Join(subdir, "target.py")
	if err := os.WriteFile(rootTarget, []byte("def root_only():\n    pass\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(subdirTarget, []byte("def Foo():\n    return 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	reg := locator.NewRegistry()
	reg.Register(locator.Location{
		FilePath:     "target.py",
		ResolvedPath: rootTarget,
		Line:         1,
		Name:         "function Foo",
	})

	output := "── function Foo (L1) in target.py [L1] ──\nDefinition:\n  1: def Foo():\n"
	result := buildCrossPatternIndexFromExecutions([]formattedPatternExecution{
		{
			Index: 0,
			singlePatternExecution: singlePatternExecution{
				Pattern: "Foo",
				Output:  output,
				Bundle: &SymbolBundle{
					Identity: SymbolBundleIdentity{
						File:        "target.py",
						Kind:        "function",
						DisplayName: "Foo",
						Line:        1,
					},
					Debug: SymbolBundleDebug{
						FileRootPath: subdir,
					},
				},
			},
		},
		{
			Index: 1,
			singlePatternExecution: singlePatternExecution{
				Pattern: "FooAlias",
				Output:  output,
				Bundle: &SymbolBundle{
					Identity: SymbolBundleIdentity{
						File:        "target.py",
						Kind:        "function",
						DisplayName: "Foo",
						Line:        1,
					},
					Debug: SymbolBundleDebug{
						FileRootPath: subdir,
					},
				},
			},
		},
	}, reg, SearchOptions{
		ProjectMapRootPath: root,
		InvocationCWD:      subdir,
	})
	if !strings.Contains(result, "target.py") || !strings.Contains(result, "[L2]") {
		t.Fatalf("expected file index locator to be registered from execution metadata, got:\n%s", result)
	}
	loc, ok := reg.Resolve("[L2]")
	if !ok {
		t.Fatal("expected file index locator [L2] to be registered")
	}
	if loc.ResolvedPath != subdirTarget {
		t.Fatalf("expected file index locator to use bundle resolved path %s, got %+v", subdirTarget, loc)
	}
}

func TestAffectedFilePathsFromExecution_BundleUsesExecutionMetadata(t *testing.T) {
	root := t.TempDir()
	subdir := filepath.Join(root, "pkg")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatal(err)
	}

	rootTarget := filepath.Join(root, "target.py")
	subdirTarget := filepath.Join(subdir, "target.py")
	helperTarget := filepath.Join(subdir, "helper.py")
	for path, content := range map[string]string{
		rootTarget:   "def root_only():\n    pass\n",
		subdirTarget: "def Foo():\n    return 1\n",
		helperTarget: "def helper():\n    return 2\n",
	} {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	execution := formattedPatternExecution{
		Index: 0,
		singlePatternExecution: singlePatternExecution{
			Pattern:       "Foo",
			Output:        "── function Foo (L1) in target.py [L1] ──\nDefinition:\n  1: def Foo():\n",
			AffectedFiles: []string{helperTarget},
			Bundle: &SymbolBundle{
				Definition: SymbolBundleDefinition{
					File: "target.py",
					Line: 1,
				},
				Identity: SymbolBundleIdentity{
					File:        "target.py",
					Kind:        "function",
					DisplayName: "Foo",
					Line:        1,
				},
				Debug: SymbolBundleDebug{
					FileRootPath: subdir,
				},
			},
		},
	}

	affected := affectedFilePathsFromExecution(execution, SearchOptions{
		ProjectMapRootPath: root,
		InvocationCWD:      subdir,
	})
	if !containsAffectedFile(affected, helperTarget) {
		t.Fatalf("expected explicit affected file %s, got %v", helperTarget, affected)
	}
	if !containsAffectedFile(affected, subdirTarget) {
		t.Fatalf("expected bundle-derived affected file %s, got %v", subdirTarget, affected)
	}
	if containsAffectedFile(affected, rootTarget) {
		t.Fatalf("did not expect stale root path %s in affected files: %v", rootTarget, affected)
	}
}
