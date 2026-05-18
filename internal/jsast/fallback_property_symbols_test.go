package jsast

import (
	"slices"
	"testing"
)

func TestFallbackSymbolsFromSourceProperties(t *testing.T) {
	src := []byte("class View {\n" +
		"  render = () => 'ok'\n" +
		"  title: string\n" +
		"}\n" +
		"interface Store {\n" +
		"  save: (id: string) => void\n" +
		"}\n" +
		"outside: () => null\n")
	parsed, err := ParseBytes("src/view.ts", src)
	if err != nil {
		t.Fatalf("ParseBytes() error = %v", err)
	}
	defer parsed.Close()

	symbols := fallbackSymbolsFromSource(parsed)
	for _, want := range []struct {
		name string
		kind string
	}{
		{"render", "field"},
		{"title", "field"},
		{"save", "property"},
	} {
		if !slices.ContainsFunc(symbols, func(symbol Symbol) bool {
			return symbol.Name == want.name && symbol.Kind == want.kind
		}) {
			t.Fatalf("symbols = %+v, want fallback %s %s", symbols, want.kind, want.name)
		}
	}
	if slices.ContainsFunc(symbols, func(symbol Symbol) bool {
		return symbol.Name == "outside"
	}) {
		t.Fatalf("symbols = %+v, want no top-level property fallback", symbols)
	}
}
