package search

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/navigation"
	"github.com/susugadx/xelyon-cli/internal/repomap"
)

func TestSearchArtifactHelpers(t *testing.T) {
	analysis := AnalyzeQuery("Builder")
	if !analysis.LooksLikeBareIdentifier || analysis.TrimmedPattern != "Builder" {
		t.Fatalf("AnalyzeQuery() = %+v, want bare identifier analysis", analysis)
	}

	if !HasMultiplePatterns("foo,bar") {
		t.Fatal("HasMultiplePatterns(foo,bar) = false, want true")
	}
	if HasMultiplePatterns(`foo\,bar`) {
		t.Fatal(`HasMultiplePatterns("foo\\,bar") = true, want false`)
	}

	if !ShouldPreferImpactIntent("Builder") {
		t.Fatal("ShouldPreferImpactIntent(Builder) = false, want true")
	}
	if !ShouldPreferImpactIntent("pkg.Builder") {
		t.Fatal("ShouldPreferImpactIntent(pkg.Builder) = false, want true")
	}
	if ShouldPreferImpactIntent("foo,bar") {
		t.Fatal("ShouldPreferImpactIntent(foo,bar) = true, want false for multi-pattern")
	}
	if ShouldPreferImpactIntent("  ") {
		t.Fatal("ShouldPreferImpactIntent(blank) = true, want false")
	}
}

func TestExecuteSearchCodeArtifactWithConfig_MultiPattern(t *testing.T) {
	setupSearchTestMocks(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "target.go")
	if err := writeSearchArtifactTestFile(path, "package main\n\nfunc Builder() {}\nfunc Runner() {}\n"); err != nil {
		t.Fatal(err)
	}

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, SearchOptions{
		Pattern: "Builder,Runner",
		Path:    dir,
	})
	if !artifact.Metadata.MultiPattern {
		t.Fatal("MultiPattern = false, want true for comma-separated query")
	}
	if !strings.Contains(artifact.Rendered, "Pattern 1/2") {
		t.Fatalf("expected multi-pattern rendered output, got:\n%s", artifact.Rendered)
	}
}

func TestExecuteSearchCodeArtifactWithConfig_StructuredImpactMetadata(t *testing.T) {
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
	var b Builder
	_ = b.Build()
}
`,
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, SearchOptions{
		Pattern: "Builder",
		Intent:  "impact",
		Path:    dir,
		ProjectMap: &repomap.ProjectMap{
			RootPath: dir,
			Files: []*repomap.FileEntry{{
				Path: "builder.go",
				Symbols: []repomap.Symbol{{
					Name:      "Builder",
					Kind:      "interface",
					Line:      3,
					EndLine:   5,
					Signature: "type Builder interface { Build() string }",
					Exported:  true,
				}},
			}},
		},
		ProjectMapRootPath: dir,
		ProjectMapStateKey: "artifact-structured-impact",
		LSPClient: &mockGoSymbolLSPClient{
			refs: []navigation.LSPLocation{
				{File: "builder_test.go", Line: 7, Character: 9, EndLine: 7, EndChar: 14},
			},
			impls: []navigation.LSPLocation{
				{File: "builder_impl.go", Line: 3, Character: 6, EndLine: 3, EndChar: 17},
			},
		},
	})

	if !artifact.Metadata.StructuredImpact {
		t.Fatal("StructuredImpact = false, want true")
	}
	if artifact.Metadata.Bundle == nil {
		t.Fatal("Bundle = nil, want structured bundle")
	}
	if artifact.Metadata.Ambiguous {
		t.Fatal("Ambiguous = true, want false for single resolved symbol")
	}
	if len(artifact.Metadata.AffectedFiles) == 0 {
		t.Fatal("AffectedFiles should not be empty for structured impact output")
	}
	if !strings.Contains(artifact.Rendered, "Related Tests") {
		t.Fatalf("expected rendered impact output to contain related tests, got:\n%s", artifact.Rendered)
	}
}

func TestExecuteSearchCodeArtifactWithConfig_StructuredImpactAmbiguous(t *testing.T) {
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
	artifact := ExecuteSearchCodeArtifactWithConfig(nil, cache, SearchOptions{
		Pattern:  "Build",
		Intent:   "impact",
		Path:     dir,
		FileType: "go",
	})

	if !artifact.Metadata.StructuredImpact {
		t.Fatal("StructuredImpact = false, want true for structured ambiguous result")
	}
	if !artifact.Metadata.Ambiguous {
		t.Fatal("Ambiguous = false, want true for multiple structured symbol matches")
	}
	if artifact.Metadata.Bundle != nil {
		t.Fatalf("Bundle = %+v, want nil for ambiguous result", artifact.Metadata.Bundle)
	}
	if len(artifact.Metadata.AffectedFiles) != 2 {
		t.Fatalf("AffectedFiles = %v, want two ambiguous candidates", artifact.Metadata.AffectedFiles)
	}
	if !strings.Contains(artifact.Rendered, `Multiple symbols matched "Build":`) {
		t.Fatalf("expected ambiguous structured output, got:\n%s", artifact.Rendered)
	}
}

func TestExecuteSearchCodeArtifactWithConfig_ImpactFallbackAddsTestProbe(t *testing.T) {
	setupSearchTestMocks(t)

	dir := setupMultiLangDir(t, map[string]string{
		"docs/builder.md":      "Builder\n",
		"docs/builder_impl.md": "BuilderImpl\n",
		"docs/tests.md":        "TestBuilder\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, SearchOptions{
		Pattern:     "Builder",
		Intent:      "impact",
		Path:        dir,
		FilePattern: "*.md",
	})
	if !artifact.Metadata.MultiPattern {
		t.Fatal("MultiPattern = false, want true for impact fallback expansion")
	}
	if !strings.Contains(artifact.Rendered, "TestBuilder") {
		t.Fatalf("expected impact fallback to add test probe, got:\n%s", artifact.Rendered)
	}
}

func writeSearchArtifactTestFile(path, content string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(content), 0o644)
}
