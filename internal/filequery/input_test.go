package filequery

import (
	"path/filepath"
	"testing"
)

func TestParseEntry_ClassifiesDirectSyntax(t *testing.T) {
	tests := []struct {
		query string
		want  SyntaxKind
	}{
		{query: "./main.go", want: SyntaxExplicitPath},
		{query: "pkg/service.go", want: SyntaxPathCandidate},
		{query: "main.go", want: SyntaxBareExtFileCandidate},
		{query: "Makefile", want: SyntaxBareNamedFileCandidate},
		{query: "pkg.Service", want: SyntaxNone},
		{query: "A or B in docs", want: SyntaxNone},
		{query: "review harness under docs", want: SyntaxNone},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			got, ok := ParseEntry(tt.query)
			if !ok {
				t.Fatalf("ParseEntry(%q) ok = false", tt.query)
			}
			if got.Syntax != tt.want {
				t.Fatalf("Syntax = %q, want %q", got.Syntax, tt.want)
			}
		})
	}
}

func TestParseEntry_PreservesDirectSyntaxForPathLikeQueries(t *testing.T) {
	absolutePathWithSpace := filepath.Join(string(filepath.Separator), "tmp", "release notes.md")
	tests := []struct {
		query string
		want  SyntaxKind
	}{
		{query: "docs/", want: SyntaxExplicitPath},
		{query: "./docs/", want: SyntaxExplicitPath},
		{query: "A or B in docs/", want: SyntaxExplicitPath},
		{query: "search results/", want: SyntaxExplicitPath},
		{query: "terms and conditions/", want: SyntaxExplicitPath},
		{query: "pkg/file.go", want: SyntaxPathCandidate},
		{query: "./release notes.md", want: SyntaxExplicitPath},
		{query: "../release notes.md", want: SyntaxExplicitPath},
		{query: absolutePathWithSpace, want: SyntaxExplicitPath},
		{query: "release notes.md:1-2", want: SyntaxExplicitPath},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			got, ok := ParseEntry(tt.query)
			if !ok {
				t.Fatalf("ParseEntry(%q) ok = false", tt.query)
			}
			if got.Syntax != tt.want {
				t.Fatalf("Syntax = %q, want %q", got.Syntax, tt.want)
			}
			if LooksLikeNaturalLanguageSearchIntent(tt.query) {
				t.Fatalf("LooksLikeNaturalLanguageSearchIntent(%q) = true, want false", tt.query)
			}
		})
	}
}

func TestLooksLikeNaturalLanguageSearchIntent(t *testing.T) {
	tests := []struct {
		query string
		want  bool
	}{
		{query: "A or B in docs", want: true},
		{query: "review harness under docs", want: true},
		{query: "search review harness", want: true},
		{query: "searching review harness", want: true},
		{query: "finding review harness", want: true},
		{query: "look for review harness", want: true},
		{query: "looking for review harness", want: true},
		{query: "./release notes.md", want: false},
		{query: "release notes.md:1-2", want: false},
		{query: "A or B in docs/", want: false},
		{query: "search results/", want: false},
		{query: "terms and conditions/", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			if got := LooksLikeNaturalLanguageSearchIntent(tt.query); got != tt.want {
				t.Fatalf("LooksLikeNaturalLanguageSearchIntent(%q) = %v, want %v", tt.query, got, tt.want)
			}
		})
	}
}

func TestContainsNaturalLanguageSearchIntentMarker(t *testing.T) {
	tests := []struct {
		query string
		want  bool
	}{
		{query: "A or B", want: true},
		{query: "A and B", want: true},
		{query: "search review harness", want: true},
		{query: "searching review harness", want: true},
		{query: "find review harness", want: true},
		{query: "finding review harness", want: true},
		{query: "look for review harness", want: true},
		{query: "looking for review harness", want: true},
		{query: "error-or-warning", want: false},
		{query: "foo/or/bar", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			if got := ContainsNaturalLanguageSearchIntentMarker(tt.query); got != tt.want {
				t.Fatalf("ContainsNaturalLanguageSearchIntentMarker(%q) = %v, want %v", tt.query, got, tt.want)
			}
		})
	}
}

func TestLeadingSearchIntentPayloadStart(t *testing.T) {
	tests := []struct {
		query string
		want  string
		ok    bool
	}{
		{query: "search foo", want: "foo", ok: true},
		{query: "search for foo", want: "foo", ok: true},
		{query: "searching foo", want: "foo", ok: true},
		{query: "searching for foo", want: "foo", ok: true},
		{query: "find foo", want: "foo", ok: true},
		{query: "finding foo", want: "foo", ok: true},
		{query: "look for foo", want: "foo", ok: true},
		{query: "looking for foo", want: "foo", ok: true},
		{query: "look foo", ok: false},
		{query: "foo search", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			start, ok := LeadingSearchIntentPayloadStart(tt.query)
			if ok != tt.ok {
				t.Fatalf("ok = %v, want %v", ok, tt.ok)
			}
			if !ok {
				return
			}
			if got := tt.query[start:]; got != tt.want {
				t.Fatalf("payload = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParsePath(t *testing.T) {
	path, start, end := ParsePath("file.go:10-20")
	if path != "file.go" || start != 10 || end != 20 {
		t.Fatalf("ParsePath() = (%q, %d, %d)", path, start, end)
	}
}
