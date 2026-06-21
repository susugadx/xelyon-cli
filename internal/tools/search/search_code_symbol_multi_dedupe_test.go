package search

import (
	"path/filepath"
	"strings"
	"testing"
)

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
