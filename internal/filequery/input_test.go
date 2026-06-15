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

func TestParseInputMultiEntryAndMalformedQueries(t *testing.T) {
	input, ok := ParseInput("./main.go:10, internal/filequery/input.go, Makefile")
	if !ok {
		t.Fatal("ParseInput(multi-entry) ok = false, want true")
	}
	if len(input.Entries) != 3 {
		t.Fatalf("entries len = %d, want 3", len(input.Entries))
	}
	if input.Entries[0].RawPath != "./main.go" ||
		input.Entries[0].StartLine != 10 ||
		input.Entries[0].Syntax != SyntaxExplicitPath {
		t.Fatalf("first entry = %#v, want explicit ./main.go:10", input.Entries[0])
	}
	if input.Entries[1].Syntax != SyntaxPathCandidate ||
		input.Entries[2].Syntax != SyntaxBareNamedFileCandidate {
		t.Fatalf("entry syntaxes = %#v, want path candidate then bare named file", input.Entries)
	}

	for _, query := range []string{"", "   ", "main.go,", ",main.go", "main.go,,README.md"} {
		t.Run(query, func(t *testing.T) {
			if input, ok := ParseInput(query); ok {
				t.Fatalf("ParseInput(%q) = (%#v, true), want false", query, input)
			}
		})
	}
}

func TestParseInputKeepsNaturalLanguageSearchIntentOutOfDirectRouting(t *testing.T) {
	input, ok := ParseInput("search provider history compaction")
	if !ok {
		t.Fatal("ParseInput(search intent) ok = false, want parsed non-direct entry")
	}
	if len(input.Entries) != 1 || input.Entries[0].Syntax != SyntaxNone {
		t.Fatalf("entries = %#v, want single SyntaxNone search intent", input.Entries)
	}
	if InputHasOnlyExplicitPathSyntax(input) ||
		InputHasOnlyCandidateDirectSyntax(input, true) ||
		InputHasOnlyDirectReadCandidates(input, true) ||
		InputHasStrictScopedDirectIntent(input) ||
		InputHasOnlyScopedExactBatchIntent(input) {
		t.Fatalf("search intent input was accepted by a direct routing helper: %#v", input)
	}
}

func TestInputAggregateRoutingContracts(t *testing.T) {
	tests := []struct {
		name             string
		query            string
		allowNamedBare   bool
		wantExplicitOnly bool
		wantCandidate    bool
		wantDirectRead   bool
		wantStrictScoped bool
		wantScopedBatch  bool
		wantContainsPath bool
	}{
		{
			name:             "explicit only",
			query:            "./main.go:1, ../README.md",
			wantExplicitOnly: true,
			wantDirectRead:   true,
			wantStrictScoped: false,
		},
		{
			name:             "candidate only exact files",
			query:            "internal/filequery/input.go, README.md",
			wantCandidate:    true,
			wantStrictScoped: true,
			wantScopedBatch:  true,
			wantContainsPath: true,
		},
		{
			name:             "direct read allows explicit plus bare file",
			query:            "./internal/filequery/input.go, README.md",
			wantDirectRead:   true,
			wantStrictScoped: false,
		},
		{
			name:             "named bare files require opt in",
			query:            "internal/filequery/input.go, Makefile",
			allowNamedBare:   true,
			wantCandidate:    true,
			wantScopedBatch:  true,
			wantStrictScoped: true,
			wantContainsPath: true,
		},
		{
			name:             "directory candidate is not exact scoped batch",
			query:            "internal/filequery",
			wantContainsPath: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input, ok := ParseInput(tt.query)
			if !ok {
				t.Fatalf("ParseInput(%q) ok = false", tt.query)
			}
			if got := InputHasOnlyExplicitPathSyntax(input); got != tt.wantExplicitOnly {
				t.Fatalf("InputHasOnlyExplicitPathSyntax() = %v, want %v", got, tt.wantExplicitOnly)
			}
			if got := InputHasOnlyCandidateDirectSyntax(input, tt.allowNamedBare); got != tt.wantCandidate {
				t.Fatalf("InputHasOnlyCandidateDirectSyntax() = %v, want %v", got, tt.wantCandidate)
			}
			if got := InputHasOnlyDirectReadCandidates(input, tt.allowNamedBare); got != tt.wantDirectRead {
				t.Fatalf("InputHasOnlyDirectReadCandidates() = %v, want %v", got, tt.wantDirectRead)
			}
			if got := InputHasStrictScopedDirectIntent(input); got != tt.wantStrictScoped {
				t.Fatalf("InputHasStrictScopedDirectIntent() = %v, want %v", got, tt.wantStrictScoped)
			}
			if got := InputHasOnlyScopedExactBatchIntent(input); got != tt.wantScopedBatch {
				t.Fatalf("InputHasOnlyScopedExactBatchIntent() = %v, want %v", got, tt.wantScopedBatch)
			}
			if got := InputContainsPathCandidateSyntax(input); got != tt.wantContainsPath {
				t.Fatalf("InputContainsPathCandidateSyntax() = %v, want %v", got, tt.wantContainsPath)
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
