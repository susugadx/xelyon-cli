package primaryrefs

import (
	"path/filepath"
	"testing"
)

func TestParseCandidateLine(t *testing.T) {
	tests := []struct {
		name string
		line string
		want Candidate
		ok   bool
	}{
		{
			name: "text search file header",
			line: "📄 src/main.go (1 match(es))",
			want: Candidate{DisplayPath: "src/main.go", Source: SourceText},
			ok:   true,
		},
		{
			name: "symbol header with locator",
			line: "── func Foo (L5) in pkg/foo.go @loc1 ──",
			want: Candidate{DisplayPath: "pkg/foo.go", Source: SourceStructuredSymbol},
			ok:   true,
		},
		{
			name: "numbered fallback with in clause",
			line: "1. function Foo (L1) in target.py [L1]",
			want: Candidate{DisplayPath: "target.py", Source: SourceInvocationRelative},
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
			got, ok := ParseCandidateLine(tt.line)
			if ok != tt.ok {
				t.Fatalf("ParseCandidateLine(%q) ok=%v, want %v", tt.line, ok, tt.ok)
			}
			if got != tt.want {
				t.Fatalf("ParseCandidateLine(%q) = %+v, want %+v", tt.line, got, tt.want)
			}
		})
	}
}

func TestCollectCandidatesPreservesOrder(t *testing.T) {
	output := "📄 src/a.go (1 match(es))\n── func Foo (L1) in src/b.go ──\nNo matches found\n"
	candidates := CollectCandidates(output)
	if len(candidates) != 2 {
		t.Fatalf("expected 2 candidates, got %d", len(candidates))
	}
	if candidates[0].DisplayPath != "src/a.go" || candidates[0].Source != SourceText {
		t.Fatalf("unexpected first candidate: %+v", candidates[0])
	}
	if candidates[1].DisplayPath != "src/b.go" || candidates[1].Source != SourceStructuredSymbol {
		t.Fatalf("unexpected second candidate: %+v", candidates[1])
	}
}

func TestResolveCandidatesDedupesByDisplayAndResolvedPath(t *testing.T) {
	dir := t.TempDir()
	candidates := []Candidate{
		{DisplayPath: "src/main.go", Source: SourceText},
		{DisplayPath: "src/main.go", Source: SourceText},
	}

	refs := ResolveCandidates(candidates, Resolver{
		Text: func(displayPath string) string {
			return filepath.Join(dir, filepath.FromSlash(displayPath))
		},
	})
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

func TestParseNumberedCandidateFilePath(t *testing.T) {
	tests := []struct {
		name string
		line string
		want string
		ok   bool
	}{
		{name: "function candidate with path first", line: "1. src/agent.go function Close (L10)", want: "src/agent.go", ok: true},
		{name: "method candidate with path first", line: "2. pkg/agent.go method (*Agent).Close (L20)", want: "pkg/agent.go", ok: true},
		{name: "invalid shape", line: "1. function Close (L10) in src/agent.go", ok: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := ParseNumberedCandidateFilePath(tt.line)
			if ok != tt.ok {
				t.Fatalf("ParseNumberedCandidateFilePath(%q) ok=%v, want %v", tt.line, ok, tt.ok)
			}
			if got != tt.want {
				t.Fatalf("ParseNumberedCandidateFilePath(%q) = %q, want %q", tt.line, got, tt.want)
			}
		})
	}
}
