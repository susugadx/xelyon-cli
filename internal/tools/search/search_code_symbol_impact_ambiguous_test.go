package search

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/navigation"
	"github.com/susugadx/xelyon-cli/internal/repomap"
)

func TestExecuteSearchCode_ImpactIntentUsesStructuredGoSingleSymbolPath(t *testing.T) {
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
		"client.go": `package example

func UseBuilder(b Builder) string {
	return b.Build()
}
`,
		"api/builder_test.go": `package api

func TestBuilderUsage(t *testing.T) {
	var b example.Builder
	_ = b.Build()
}
`,
	})

	output := ExecuteSearchCode(SearchOptions{
		Pattern: "Builder",
		Intent:  "impact",
		Path:    dir,
		ProjectMap: &repomap.ProjectMap{
			RootPath: dir,
			Files: []*repomap.FileEntry{
				{
					Path: "builder.go",
					Symbols: []repomap.Symbol{
						{Name: "Builder", Kind: "interface", Line: 3, EndLine: 5, Signature: "type Builder interface { Build() string }", Exported: true},
					},
				},
			},
		},
		ProjectMapRootPath: dir,
		ProjectMapStateKey: "structured-impact-go",
		LSPClient: &mockGoSymbolLSPClient{
			refs: []navigation.LSPLocation{
				{File: "client.go", Line: 4, Character: 9, EndLine: 4, EndChar: 14},
				{File: "api/builder_test.go", Line: 5, Character: 8, EndLine: 5, EndChar: 13},
			},
			impls: []navigation.LSPLocation{
				{File: "builder_impl.go", Line: 3, Character: 6, EndLine: 3, EndChar: 17},
			},
		},
	})

	if strings.Contains(output, "Pattern 1/") {
		t.Fatalf("expected structured impact path instead of fake multi-pattern output, got:\n%s", output)
	}
	if !strings.Contains(output, "Risk: high") {
		t.Fatalf("expected high risk in structured impact output, got:\n%s", output)
	}
	if !strings.Contains(output, "Recommended reads:") {
		t.Fatalf("expected recommended reads in structured impact output, got:\n%s", output)
	}
	if !strings.Contains(output, "Related Implementations") {
		t.Fatalf("expected implementations section in structured impact output, got:\n%s", output)
	}
}

func TestExecuteSearchCode_ImpactIntentStructuredGoPreservesTestNameProbeFallback(t *testing.T) {
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

import "testing"

func TestBuilder(t *testing.T) {
	var impl FileBuilder
	_ = impl.Build()
}
`,
	})

	output := ExecuteSearchCode(SearchOptions{
		Pattern: "Builder",
		Intent:  "impact",
		Path:    dir,
		ProjectMap: &repomap.ProjectMap{
			RootPath: dir,
			Files: []*repomap.FileEntry{
				{
					Path: "builder.go",
					Symbols: []repomap.Symbol{
						{Name: "Builder", Kind: "interface", Line: 3, EndLine: 5, Signature: "type Builder interface { Build() string }", Exported: true},
					},
				},
			},
		},
		ProjectMapRootPath: dir,
		ProjectMapStateKey: "structured-impact-go-test-probe",
		LSPClient: &mockGoSymbolLSPClient{
			refs: []navigation.LSPLocation{
				{File: "builder_impl.go", Line: 5, Character: 9, EndLine: 5, EndChar: 14},
			},
			impls: []navigation.LSPLocation{
				{File: "builder_impl.go", Line: 3, Character: 6, EndLine: 3, EndChar: 17},
			},
		},
	})

	if !strings.Contains(output, "Related Tests") || !strings.Contains(output, "TestBuilder") {
		t.Fatalf("expected structured impact output to preserve Test<Symbol> probe fallback, got:\n%s", output)
	}
}

func TestExecuteSearchCode_ImpactIntentStructuredGoKeepsAmbiguousCandidates(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"build.go": `package example

func Build() string { return "" }
`,
		"config.go": `package example

type Config struct{}

func (Config) Build() string { return "" }
`,
	})

	cache := &testSearchCache{data: make(map[string]string)}
	opts := SearchOptions{
		Pattern:  "Build",
		Intent:   "impact",
		Path:     dir,
		FileType: "go",
	}

	output1 := ExecuteSearchCodeWithCache(cache, opts)
	if !strings.Contains(output1, `Multiple symbols matched "Build":`) {
		t.Fatalf("expected structured candidate list for ambiguous impact symbol, got:\n%s", output1)
	}
	if strings.Contains(output1, "Pattern 1/") {
		t.Fatalf("expected structured impact path to avoid fake multi-pattern fallback, got:\n%s", output1)
	}

	searchKey := singlePatternBundleCacheKey("Build", cache.lastSetPath)
	affected := cache.affected[searchKey]
	for _, want := range []string{filepath.Join(dir, "build.go"), filepath.Join(dir, "config.go")} {
		if !containsAffectedFile(affected, want) {
			t.Fatalf("expected structured ambiguous cache to track %s, got %v", want, affected)
		}
	}

	getCalls := cache.getCalls
	output2 := ExecuteSearchCodeWithCache(cache, opts)
	if cache.getCalls <= getCalls {
		t.Fatal("expected cache lookup on repeated structured ambiguous impact search")
	}
	if output2 != output1 {
		t.Fatalf("expected cached ambiguous structured impact output to be stable,\nfirst:\n%s\nsecond:\n%s", output1, output2)
	}
}
