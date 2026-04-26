package filequery

import "testing"

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

func TestParsePath(t *testing.T) {
	path, start, end := ParsePath("file.go:10-20")
	if path != "file.go" || start != 10 || end != 20 {
		t.Fatalf("ParsePath() = (%q, %d, %d)", path, start, end)
	}
}
