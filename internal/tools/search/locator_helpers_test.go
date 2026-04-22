package search

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParsePrimaryFileRefCandidateLine(t *testing.T) {
	tests := []struct {
		name string
		line string
		want primaryFileRefCandidate
		ok   bool
	}{
		{
			name: "text search file header",
			line: "📄 src/main.go (1 match(es))",
			want: primaryFileRefCandidate{DisplayPath: "src/main.go", Source: primaryFileRefSourceText},
			ok:   true,
		},
		{
			name: "symbol header with locator",
			line: "── func Foo (L5) in pkg/foo.go @loc1 ──",
			want: primaryFileRefCandidate{DisplayPath: "pkg/foo.go", Source: primaryFileRefSourceStructuredSymbol},
			ok:   true,
		},
		{
			name: "numbered fallback with in clause",
			line: "1. function Foo (L1) in target.py [L1]",
			want: primaryFileRefCandidate{DisplayPath: "target.py", Source: primaryFileRefSourceInvocationRelative},
			ok:   true,
		},
		{
			name: "invalid line",
			line: "No matches found",
			ok:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parsePrimaryFileRefCandidateLine(tt.line)
			if ok != tt.ok {
				t.Fatalf("parsePrimaryFileRefCandidateLine(%q) ok=%v, want %v", tt.line, ok, tt.ok)
			}
			if got != tt.want {
				t.Fatalf("parsePrimaryFileRefCandidateLine(%q) = %+v, want %+v", tt.line, got, tt.want)
			}
		})
	}
}

func TestParseTextPrimaryFileRefCandidate(t *testing.T) {
	got, ok := parseTextPrimaryFileRefCandidate("📄 src/main.go (1 match(es))")
	if !ok {
		t.Fatal("expected text candidate to be parsed")
	}
	want := primaryFileRefCandidate{DisplayPath: "src/main.go", Source: primaryFileRefSourceText}
	if got != want {
		t.Fatalf("unexpected candidate: got %+v, want %+v", got, want)
	}

	if _, ok := parseTextPrimaryFileRefCandidate("src/main.go"); ok {
		t.Fatal("expected non-text line to be rejected")
	}
}

func TestParseStructuredHeaderPrimaryFileRefCandidate(t *testing.T) {
	got, ok := parseStructuredHeaderPrimaryFileRefCandidate("── func Foo (L5) in pkg/foo.go @loc1 ──")
	if !ok {
		t.Fatal("expected structured header to be parsed")
	}
	want := primaryFileRefCandidate{DisplayPath: "pkg/foo.go", Source: primaryFileRefSourceStructuredSymbol}
	if got != want {
		t.Fatalf("unexpected candidate: got %+v, want %+v", got, want)
	}

	if _, ok := parseStructuredHeaderPrimaryFileRefCandidate("── malformed header"); ok {
		t.Fatal("expected malformed structured header to be rejected")
	}
}

func TestParseNumberedPrimaryFileRefCandidate(t *testing.T) {
	t.Run("kind-token line", func(t *testing.T) {
		got, ok := parseNumberedPrimaryFileRefCandidate("1. src/agent.go function Close (L10)")
		if !ok {
			t.Fatal("expected numbered kind-token line to be parsed")
		}
		want := primaryFileRefCandidate{DisplayPath: "src/agent.go", Source: primaryFileRefSourceStructuredSymbol}
		if got != want {
			t.Fatalf("unexpected candidate: got %+v, want %+v", got, want)
		}
	})

	t.Run("in-clause fallback", func(t *testing.T) {
		got, ok := parseNumberedPrimaryFileRefCandidate("1. function Foo (L1) in target.py [L1]")
		if !ok {
			t.Fatal("expected numbered fallback line to be parsed")
		}
		want := primaryFileRefCandidate{DisplayPath: "target.py", Source: primaryFileRefSourceInvocationRelative}
		if got != want {
			t.Fatalf("unexpected candidate: got %+v, want %+v", got, want)
		}
	})

	t.Run("invalid line", func(t *testing.T) {
		if _, ok := parseNumberedPrimaryFileRefCandidate("no numbered line"); ok {
			t.Fatal("expected invalid line to be rejected")
		}
	})
}

func TestCollectPrimaryFileRefCandidates_PreservesOrder(t *testing.T) {
	output := "📄 src/a.go (1 match(es))\n── func Foo (L1) in src/b.go ──\nNo matches found\n"
	candidates := collectPrimaryFileRefCandidates(output)
	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(candidates))
	}
	if candidates[0].DisplayPath != "src/a.go" || candidates[0].Source != primaryFileRefSourceText {
		t.Fatalf("unexpected first candidate: %+v", candidates[0])
	}
	if candidates[1].DisplayPath != "src/b.go" || candidates[1].Source != primaryFileRefSourceStructuredSymbol {
		t.Fatalf("unexpected second candidate: %+v", candidates[1])
	}
}

func TestResolvePrimaryFileRefs_DedupByDisplayAndResolvedPath(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "src"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "src", "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	candidates := []primaryFileRefCandidate{
		{DisplayPath: "src/main.go", Source: primaryFileRefSourceText},
		{DisplayPath: "src/main.go", Source: primaryFileRefSourceText},
	}

	refs := resolvePrimaryFileRefs(candidates, SearchOptions{Path: dir})
	if len(refs) != 1 {
		t.Fatalf("expected 1 deduped ref, got %d (%+v)", len(refs), refs)
	}
	if refs[0].DisplayPath != "src/main.go" {
		t.Fatalf("unexpected display path: %+v", refs[0])
	}
	if refs[0].ResolvedPath != filepath.Join(dir, "src", "main.go") {
		t.Fatalf("unexpected resolved path: %+v", refs[0])
	}
}

func TestPrimaryFileRefResolverResolveCandidate_RejectsEmptyDisplayPath(t *testing.T) {
	resolver := newPrimaryFileRefResolver(SearchOptions{})
	if _, ok := resolver.resolveCandidate(primaryFileRefCandidate{DisplayPath: "   ", Source: primaryFileRefSourceText}); ok {
		t.Fatal("expected empty display path candidate to be rejected")
	}
}

func TestPrimaryFileRefCandidateCollectorAddLine(t *testing.T) {
	collector := newPrimaryFileRefCandidateCollector()
	collector.addLine("No matches found")
	collector.addLine("📄 src/main.go (1 match(es))")
	collector.addLine("📄 src/main.go (1 match(es))")

	got := collector.results()
	if len(got) != 2 {
		t.Fatalf("expected 2 collected candidates, got %d (%+v)", len(got), got)
	}
	if got[0].DisplayPath != "src/main.go" || got[0].Source != primaryFileRefSourceText {
		t.Fatalf("unexpected first candidate: %+v", got[0])
	}
	if got[1].DisplayPath != "src/main.go" || got[1].Source != primaryFileRefSourceText {
		t.Fatalf("unexpected second candidate: %+v", got[1])
	}
}

func TestPrimaryFileRefCandidateCollectorAddCandidate_RejectsEmptyDisplayPath(t *testing.T) {
	collector := newPrimaryFileRefCandidateCollector()
	collector.addCandidate(primaryFileRefCandidate{DisplayPath: "   ", Source: primaryFileRefSourceText})
	if got := collector.results(); len(got) != 0 {
		t.Fatalf("expected empty candidate to be rejected, got %+v", got)
	}
}

func TestPrimaryFileRefLineParsers_DefaultOrder(t *testing.T) {
	if len(primaryFileRefLineParsers) != 3 {
		t.Fatalf("expected exactly 3 default parsers, got %d", len(primaryFileRefLineParsers))
	}
	if _, ok := primaryFileRefLineParsers[0]("📄 src/main.go (1 match(es))"); !ok {
		t.Fatal("expected first parser to parse text header")
	}
	if _, ok := primaryFileRefLineParsers[0]("── func Foo (L1) in src/foo.go ──"); ok {
		t.Fatal("expected first parser to reject structured header")
	}

	if _, ok := primaryFileRefLineParsers[1]("── func Foo (L1) in src/foo.go ──"); !ok {
		t.Fatal("expected second parser to parse structured header")
	}
	if _, ok := primaryFileRefLineParsers[1]("1. src/foo.go function Foo (L1)"); ok {
		t.Fatal("expected second parser to reject numbered line")
	}

	if _, ok := primaryFileRefLineParsers[2]("1. src/foo.go function Foo (L1)"); !ok {
		t.Fatal("expected third parser to parse numbered line")
	}
}

func TestHasNumericListPrefix(t *testing.T) {
	tests := []struct {
		line string
		want bool
	}{
		{line: "1. something", want: true},
		{line: "12. in src/main.go", want: true},
		{line: "1) invalid", want: false},
		{line: "foo", want: false},
		{line: "", want: false},
	}

	for _, tt := range tests {
		got := hasNumericListPrefix(tt.line)
		if got != tt.want {
			t.Fatalf("hasNumericListPrefix(%q) = %v, want %v", tt.line, got, tt.want)
		}
	}
}

func TestParseNumberedCandidateFilePath(t *testing.T) {
	tests := []struct {
		name string
		line string
		want string
		ok   bool
	}{
		{
			name: "function candidate with path first",
			line: "1. src/agent.go function Close (L10)",
			want: "src/agent.go",
			ok:   true,
		},
		{
			name: "method candidate with path first",
			line: "2. pkg/agent.go method (*Agent).Close (L20)",
			want: "pkg/agent.go",
			ok:   true,
		},
		{
			name: "type candidate with path first",
			line: "3. pkg/model.go type Config (L30)",
			want: "pkg/model.go",
			ok:   true,
		},
		{
			name: "invalid shape",
			line: "1. function Close (L10) in src/agent.go",
			want: "",
			ok:   false,
		},
		{
			name: "unknown kind token is rejected",
			line: "4. pkg/service.go trait Service (L40)",
			want: "",
			ok:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := parseNumberedCandidateFilePath(tt.line)
			if ok != tt.ok {
				t.Fatalf("parseNumberedCandidateFilePath(%q) ok=%v, want %v", tt.line, ok, tt.ok)
			}
			if got != tt.want {
				t.Fatalf("parseNumberedCandidateFilePath(%q) = %q, want %q", tt.line, got, tt.want)
			}
		})
	}
}

func TestSplitNumberedListLinePrefix(t *testing.T) {
	rest, ok := splitNumberedListLinePrefix("12. src/main.go function Run (L10)")
	if !ok {
		t.Fatal("expected valid numbered list line")
	}
	if rest != "src/main.go function Run (L10)" {
		t.Fatalf("unexpected rest: %q", rest)
	}

	if _, ok := splitNumberedListLinePrefix("12) invalid format"); ok {
		t.Fatal("expected invalid format to be rejected")
	}
}
