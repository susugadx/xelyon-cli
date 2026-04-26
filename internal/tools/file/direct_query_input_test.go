package file

import "testing"

func TestParseDirectQueryEntryInput_SyntaxClassification(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  directQuerySyntaxKind
	}{
		{name: "sample go file stays bare ext-file candidate", query: "sample.go", want: directQuerySyntaxBareExtFileCandidate},
		{name: "elixir source file stays bare ext-file candidate", query: "main.ex", want: directQuerySyntaxBareExtFileCandidate},
		{name: "elixir script file stays bare ext-file candidate", query: "main.exs", want: directQuerySyntaxBareExtFileCandidate},
		{name: "python stub file stays bare ext-file candidate", query: "types.pyi", want: directQuerySyntaxBareExtFileCandidate},
		{name: "explicit relative path stays explicit", query: "./sample.go", want: directQuerySyntaxExplicitPath},
		{name: "directory syntax stays explicit", query: "config/", want: directQuerySyntaxExplicitPath},
		{name: "slash directory syntax stays explicit", query: "internal/agent/", want: directQuerySyntaxExplicitPath},
		{name: "explicit relative slash directory stays explicit", query: "./internal/agent", want: directQuerySyntaxExplicitPath},
		{name: "explicit line range stays explicit", query: "sample.go:10-20", want: directQuerySyntaxExplicitPath},
		{name: "slash file path stays path candidate", query: "pkg/errors.go", want: directQuerySyntaxPathCandidate},
		{name: "slash file range stays explicit", query: "pkg/errors.go:1-10", want: directQuerySyntaxExplicitPath},
		{name: "slash package literal stays searchable syntax", query: "pkg/errors", want: directQuerySyntaxNone},
		{name: "import-like slash literal stays searchable syntax", query: "github.com/foo/bar", want: directQuerySyntaxNone},
		{name: "makefile stays bare named-file candidate", query: "Makefile", want: directQuerySyntaxBareNamedFileCandidate},
		{name: "directory name does not become bare file candidate", query: "config", want: directQuerySyntaxNone},
		{name: "qualified package symbol is not direct syntax", query: "pkg.Func", want: directQuerySyntaxNone},
		{name: "qualified method symbol is not direct syntax", query: "Builder.Build", want: directQuerySyntaxNone},
		{name: "module-style dotted symbol is not direct syntax", query: "MyModule.Version", want: directQuerySyntaxNone},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input, ok := parseDirectQueryEntryInput(tt.query)
			if !ok {
				t.Fatalf("parseDirectQueryEntryInput(%q) returned ok=false", tt.query)
			}
			if input.Syntax != tt.want {
				t.Fatalf("parseDirectQueryEntryInput(%q).Syntax = %q, want %q", tt.query, input.Syntax, tt.want)
			}
		})
	}
}

func TestLooksLikeExplicitDirectQuery(t *testing.T) {
	tests := []struct {
		query string
		want  bool
	}{
		{query: "sample.go", want: false},
		{query: "sample.go:10-20", want: true},
		{query: "internal/tools", want: false},
		{query: "internal/tools/", want: true},
		{query: "./internal/tools", want: true},
		{query: "pkg/errors", want: false},
		{query: "pkg/errors.go", want: false},
		{query: "pkg/errors.go:1-10", want: true},
		{query: "github.com/foo/bar", want: false},
		{query: "./config", want: true},
		{query: "pkg.Func", want: false},
		{query: "Builder.Build", want: false},
		{query: "MyModule.Version", want: false},
	}

	for _, tt := range tests {
		if got := looksLikeExplicitDirectQuery(tt.query); got != tt.want {
			t.Fatalf("looksLikeExplicitDirectQuery(%q) = %v, want %v", tt.query, got, tt.want)
		}
	}
}
