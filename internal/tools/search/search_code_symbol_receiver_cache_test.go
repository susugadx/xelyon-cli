package search

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/navigation"
	"github.com/susugadx/xelyon-cli/internal/searchcache"
)

func TestStructuredGoImpactMethodProbeReceiverCachesClearOnSearchCacheInvalidation(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"config.go": `package example

type Config struct{}

func (Config) Build() string { return "" }
`,
		"config_test.go": `package example

import (
	"testing"

	otherpkg "example/otherpkg"
)

func TestBuild(t *testing.T) {
	var b otherpkg.Builder
	_ = b.Build()
}
`,
		"otherpkg/builder.go": `package otherpkg

type Builder struct{}

func (Builder) Build() string { return "" }
`,
	})

	testPath := filepath.Join(dir, "config_test.go")
	testSrc, err := os.ReadFile(testPath)
	if err != nil {
		t.Fatal(err)
	}
	opts := SearchOptions{
		Path:               dir,
		ProjectMapRootPath: dir,
	}

	if role := methodProbeQualifiedReceiverRole("otherpkg.Builder", testPath, testSrc, opts); role != methodProbeReceiverRoleConcrete {
		t.Fatalf("expected initial qualified receiver role to be concrete, got %q", role)
	}
	if !methodProbeQualifiedReceiverHasDirectMethod("otherpkg.Builder", "Build", testPath, testSrc, opts) {
		t.Fatal("expected initial qualified receiver direct-method probe to succeed")
	}

	if err := os.WriteFile(filepath.Join(dir, "otherpkg", "builder.go"), []byte(`package otherpkg

type Builder interface {
	Build() string
}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	searchcache.NotifySearchCacheInvalidatedKeys([]string{"structured-go-impact-method-probe"})

	if role := methodProbeQualifiedReceiverRole("otherpkg.Builder", testPath, testSrc, opts); role != methodProbeReceiverRoleInterface {
		t.Fatalf("expected invalidation to clear receiver role cache, got %q", role)
	}
	if methodProbeQualifiedReceiverHasDirectMethod("otherpkg.Builder", "Build", testPath, testSrc, opts) {
		t.Fatal("expected invalidation to clear direct-method cache after receiver changed to interface")
	}
}

func TestStructuredGoImpactMethodProbeTracksDependencyFilesForSearchCacheInvalidation(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"config.go": `package example

type Config struct{}

func (Config) Build() string { return "" }
`,
		"config_test.go": `package example

import (
	"testing"

	otherpkg "example/otherpkg"
)

func TestBuild(t *testing.T) {
	var b otherpkg.Builder
	_ = b.Build()
}
`,
		"otherpkg/builder.go": `package otherpkg

type Builder struct{}

func (Builder) Build() string { return "" }
`,
	})

	cache := &testSearchCache{data: make(map[string]string)}
	result := ExecuteSearchCodeWithCache(cache, SearchOptions{
		Pattern:   "Config.Build",
		Intent:    "impact",
		Path:      dir,
		FileType:  "go",
		LSPClient: &mockGoSymbolLSPClient{refs: []navigation.LSPLocation{{File: "config.go", Line: 5, Character: 11, EndLine: 5, EndChar: 16}}},
	})
	if !strings.Contains(result, "Config.Build") && !strings.Contains(result, "Config") {
		t.Fatalf("expected structured go impact output, got:\n%s", result)
	}

	want := filepath.Join(dir, "otherpkg", "builder.go")
	searchKey := singlePatternBundleCacheKey("Config.Build", cache.lastSetPath)
	if !containsAffectedFile(cache.affected[searchKey], want) {
		t.Fatalf("expected probe dependency %s to be tracked in affected files, got %v", want, cache.affected[searchKey])
	}

	cache.InvalidateSearchCacheForFile(want)
	if _, ok := cache.GetSearch("Config.Build", cache.lastSetPath); ok {
		t.Fatalf("expected search cache entry to be invalidated after editing probe dependency %s", want)
	}
}
