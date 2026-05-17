package jsast

import (
	"slices"
	"testing"
)

func TestFallbackSymbolsFromSourceMethods(t *testing.T) {
	src := []byte("class View {\n" +
		"  render() { return 'ok' }\n" +
		"  nested() {\n" +
		"    if (ready) { return render() }\n" +
		"  }\n" +
		"}\n" +
		"render()\n")
	parsed, err := ParseBytes("src/view.ts", src)
	if err != nil {
		t.Fatalf("ParseBytes() error = %v", err)
	}
	defer parsed.Close()

	symbols := fallbackSymbolsFromSource(parsed)
	for _, want := range []string{"render", "nested"} {
		if !slices.ContainsFunc(symbols, func(symbol Symbol) bool {
			return symbol.Name == want && symbol.Kind == "method"
		}) {
			t.Fatalf("symbols = %+v, want fallback method %s", symbols, want)
		}
	}
	if slices.ContainsFunc(symbols, func(symbol Symbol) bool {
		return symbol.Name == "if" || symbol.Signature == "render()"
	}) {
		t.Fatalf("symbols = %+v, want only direct type-body methods", symbols)
	}
}

func TestFallbackSymbolsFromSourceMethodsIgnoreLiteralBraces(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{
			name: "string",
			src: "class View {\n" +
				"  render() { return \"{\" }\n" +
				"}\n" +
				"foo()\n",
		},
		{
			name: "comment",
			src: "class View {\n" +
				"  render() { /* { */ return 'ok' }\n" +
				"}\n" +
				"foo()\n",
		},
		{
			name: "regex",
			src: "class View {\n" +
				"  render() { const r = /}/; return r }\n" +
				"}\n" +
				"foo()\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := ParseBytes("src/view.ts", []byte(tt.src))
			if err != nil {
				t.Fatalf("ParseBytes() error = %v", err)
			}
			defer parsed.Close()

			symbols := fallbackSymbolsFromSource(parsed)
			if slices.ContainsFunc(symbols, func(symbol Symbol) bool {
				return symbol.Name == "foo" && symbol.Kind == "method"
			}) {
				t.Fatalf("symbols = %+v, want top-level foo() outside fallback type body", symbols)
			}
		})
	}
}

func TestExtractSymbolsDoesNotAddFallbackMethodsForUsableTrees(t *testing.T) {
	src := []byte("class View {\n" +
		"  render() { const r = /}/; return r }\n" +
		"}\n" +
		"foo()\n")
	symbols, err := ExtractSymbols("src/view.ts", src)
	if err != nil {
		t.Fatalf("ExtractSymbols() error = %v", err)
	}

	if slices.ContainsFunc(symbols, func(symbol Symbol) bool {
		return symbol.Name == "foo" && symbol.Kind == "method"
	}) {
		t.Fatalf("symbols = %+v, want no fallback method for top-level foo()", symbols)
	}
}
