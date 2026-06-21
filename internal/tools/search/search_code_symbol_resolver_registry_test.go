package search

import "testing"

func TestResolverForLanguageDispatchesSupportedLanguages(t *testing.T) {
	tests := []struct {
		lang        string
		wantGo      bool
		wantGeneric bool
		wantSpec    string
	}{
		{lang: "go", wantGo: true},
		{lang: "js", wantGeneric: true, wantSpec: "js"},
		{lang: "python", wantGeneric: true, wantSpec: "python"},
		{lang: "rust", wantGeneric: true, wantSpec: "rust"},
		{lang: "java", wantGeneric: true, wantSpec: "java"},
		{lang: "csharp", wantGeneric: true, wantSpec: "csharp"},
		{lang: "php", wantGeneric: true, wantSpec: "php"},
		{lang: "ruby", wantGeneric: true, wantSpec: "ruby"},
		{lang: "swift", wantGeneric: true, wantSpec: "swift"},
		{lang: "scala", wantGeneric: true, wantSpec: "scala"},
		{lang: "elixir", wantGeneric: true, wantSpec: "elixir"},
		{lang: "lua", wantGeneric: true, wantSpec: "lua"},
		{lang: "cpp", wantGeneric: true, wantSpec: "cpp"},
		{lang: "", wantGeneric: true, wantSpec: ""},
	}

	for _, tt := range tests {
		t.Run(tt.lang, func(t *testing.T) {
			resolver := resolverForLanguage(tt.lang)
			if resolver == nil {
				t.Fatal("resolverForLanguage() = nil, want resolver")
			}
			if tt.wantGo {
				if _, ok := resolver.(goSymbolResolver); !ok {
					t.Fatalf("resolverForLanguage(%q) = %T, want goSymbolResolver", tt.lang, resolver)
				}
				return
			}
			got, ok := resolver.(genericLanguageResolver)
			if !ok || !tt.wantGeneric {
				t.Fatalf("resolverForLanguage(%q) = %T, want genericLanguageResolver", tt.lang, resolver)
			}
			if got.spec.language != tt.wantSpec {
				t.Fatalf("resolver spec language = %q, want %q", got.spec.language, tt.wantSpec)
			}
			if got.spec.resolve == nil {
				t.Fatal("resolver spec resolve = nil")
			}
		})
	}
}

func TestResolverForLanguageUnknownIsUnsupported(t *testing.T) {
	if got := resolverForLanguage("unknown"); got != nil {
		t.Fatalf("resolverForLanguage(unknown) = %T, want nil", got)
	}
}
