package search

import (
	"strings"
	"testing"
)

func TestEffectiveSearchPatterns_LiteralPatternInputPreservesComma(t *testing.T) {
	opts := SearchOptions{
		PatternInput: NewLiteralPatternInput("foo,bar"),
	}
	opts = applyPatternInput(opts)

	got := effectiveSearchPatterns(opts)
	if len(got) != 1 || got[0] != "foo,bar" {
		t.Fatalf("effectiveSearchPatterns = %#v, want single literal comma pattern", got)
	}
}

func TestEffectiveSearchPatterns_DefaultPatternKeepsDelimitedCompatibility(t *testing.T) {
	got := effectiveSearchPatterns(SearchOptions{Pattern: "foo,bar"})
	if len(got) != 2 || got[0] != "foo" || got[1] != "bar" {
		t.Fatalf("effectiveSearchPatterns = %#v, want comma-delimited patterns", got)
	}
}

func TestEffectiveSearchPatterns_DelimitedPatternInputKeepsCommaSplit(t *testing.T) {
	opts := SearchOptions{
		PatternInput: NewDelimitedPatternInput("foo,bar"),
	}
	opts = applyPatternInput(opts)

	got := effectiveSearchPatterns(opts)
	if len(got) != 2 || got[0] != "foo" || got[1] != "bar" {
		t.Fatalf("effectiveSearchPatterns = %#v, want comma-delimited patterns", got)
	}
}

func TestPrepareSearchOptions_LiteralPatternInputForcesLiteralMode(t *testing.T) {
	opts, errResult := prepareSearchOptionsForRuntime(nil, SearchOptions{
		Pattern:      "ignored",
		PatternInput: NewLiteralPatternInput("Builder"),
		Mode:         string(SearchModeAuto),
		Intent:       "impact",
		IsRegex:      true,
	})
	if errResult != "" {
		t.Fatalf("unexpected prepare error: %q", errResult)
	}
	if opts.Pattern != "Builder" {
		t.Fatalf("Pattern = %q, want literal input value", opts.Pattern)
	}
	if opts.Mode != string(SearchModeLiteral) {
		t.Fatalf("Mode = %q, want literal", opts.Mode)
	}
	if opts.IsRegex {
		t.Fatal("literal pattern input should clear regex mode")
	}
	if opts.Intent != "" {
		t.Fatalf("Intent = %q, want cleared intent", opts.Intent)
	}
}

func TestExecuteSearchCode_LiteralPatternInputSkipsSymbolResolver(t *testing.T) {
	setupSearchTestMocks(t)
	dir := setupMultiLangDir(t, map[string]string{
		"builder.go": "package sample\n\nfunc Builder() {}\n",
	})

	artifact := ExecuteSearchCodeArtifactWithConfig(nil, nil, SearchOptions{
		PatternInput: NewLiteralPatternInput("Builder"),
		Path:         dir,
		FileType:     "go",
	})

	if artifact.Metadata.Bundle != nil {
		t.Fatalf("literal pattern input returned symbol bundle: %+v", artifact.Metadata.Bundle.Debug.Route)
	}
	if !strings.Contains(artifact.Rendered, "builder.go") || !strings.Contains(artifact.Rendered, "func Builder()") {
		t.Fatalf("expected literal text result for Builder, got:\n%s", artifact.Rendered)
	}
}

func TestExecuteSearchCode_LiteralPatternInputPreservesComma(t *testing.T) {
	setupSearchTestMocks(t)
	dir := setupMultiLangDir(t, map[string]string{
		"literal.txt": "foo,bar\n",
		"split.txt":   "foo only\n",
	})

	result := ExecuteSearchCode(SearchOptions{
		PatternInput: NewLiteralPatternInput("foo,bar"),
		Path:         dir,
	})

	if !strings.Contains(result, "literal.txt") {
		t.Fatalf("expected literal comma match, got:\n%s", result)
	}
	if strings.Contains(result, "split.txt") {
		t.Fatalf("literal comma pattern should not match split pattern file, got:\n%s", result)
	}
}
