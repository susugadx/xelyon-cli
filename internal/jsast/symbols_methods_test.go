package jsast

import (
	"slices"
	"testing"
)

func TestExtractSymbolsMethods(t *testing.T) {
	tests := []struct {
		name string
		path string
		src  string
		want []struct {
			name string
			kind string
		}
	}{
		{
			name: "class method",
			path: "src/service.ts",
			src: `
class UserService {
  buildUser(id: string) { return id }
}
`,
			want: []struct {
				name string
				kind string
			}{
				{"buildUser", "method"},
			},
		},
		{
			name: "interface method signature",
			path: "src/store.ts",
			src: `
interface UserStore {
  saveUser(id: string): void
}
`,
			want: []struct {
				name string
				kind string
			}{
				{"saveUser", "method"},
			},
		},
		{
			name: "mixed class and interface methods",
			path: "src/mixed.ts",
			src: `
class UserService {
  buildUser(id: string) { return id }
}
interface UserStore {
  saveUser(id: string): void
}
`,
			want: []struct {
				name string
				kind string
			}{
				{"buildUser", "method"},
				{"saveUser", "method"},
			},
		},
		{
			name: "type alias method signature",
			path: "src/store.ts",
			src: `
type UserStore = {
  saveUser(id: string): void
}
`,
			want: []struct {
				name string
				kind string
			}{
				{"saveUser", "method"},
			},
		},
		{
			name: "intersection type alias method signature",
			path: "src/store.ts",
			src: `
type BaseStore = { id: string }
type UserStore = BaseStore & {
  saveUser(id: string): void
}
`,
			want: []struct {
				name string
				kind string
			}{
				{"saveUser", "method"},
			},
		},
		{
			name: "parenthesized type alias method signature",
			path: "src/store.ts",
			src: `
type UserStore = ({
  saveUser(id: string): void
})
`,
			want: []struct {
				name string
				kind string
			}{
				{"saveUser", "method"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			symbols, err := ExtractSymbols(tt.path, []byte(tt.src))
			if err != nil {
				t.Fatalf("ExtractSymbols() error = %v", err)
			}

			for _, want := range tt.want {
				if !slices.ContainsFunc(symbols, func(symbol Symbol) bool {
					return symbol.Name == want.name && symbol.Kind == want.kind
				}) {
					t.Fatalf("symbols = %+v, want %s %s", symbols, want.kind, want.name)
				}
			}
		})
	}
}

func TestExtractSymbolsPrivateAndProtectedMethodsAreNotExported(t *testing.T) {
	src := []byte(`
export class UserStore {
  public saveUser(id: string) { return id }
  private buildCache() { return null }
  protected resetCache() { return null }
  #secret() { return null }
}
`)

	symbols, err := ExtractSymbols("src/store.ts", src)
	if err != nil {
		t.Fatalf("ExtractSymbols() error = %v", err)
	}

	assertMethodExported := func(name string, want bool) {
		t.Helper()
		for _, symbol := range symbols {
			if symbol.Name == name && symbol.Kind == "method" {
				if symbol.Exported != want {
					t.Fatalf("%s Exported = %v, want %v; symbols = %+v", name, symbol.Exported, want, symbols)
				}
				return
			}
		}
		t.Fatalf("symbols = %+v, want method %s", symbols, name)
	}
	assertMethodExported("saveUser", true)
	assertMethodExported("buildCache", false)
	assertMethodExported("resetCache", false)
	assertMethodExported("#secret", false)
}

func TestExtractSymbolsSkipsInlineObjectTypeMethodsAsTopLevelSymbols(t *testing.T) {
	src := []byte(`
export function buildUser(id: string) { return id }
function useStore(store: {
  buildUser(id: string): void
}) {}
`)

	symbols, err := ExtractSymbols("src/build.ts", src)
	if err != nil {
		t.Fatalf("ExtractSymbols() error = %v", err)
	}

	var matches []Symbol
	for _, symbol := range symbols {
		if symbol.Name == "buildUser" {
			matches = append(matches, symbol)
		}
	}
	if len(matches) != 1 {
		t.Fatalf("buildUser symbols = %+v, want one top-level function symbol", matches)
	}
	if matches[0].Kind != "function" {
		t.Fatalf("buildUser kind = %q, want function", matches[0].Kind)
	}
}

func TestExtractSymbolsSkipsObjectLiteralMethodsAsTopLevelSymbols(t *testing.T) {
	src := []byte(`
export function buildUser(id: string) { return id }
export const handlers = {
  buildUser(id: string) { return id }
}
`)

	symbols, err := ExtractSymbols("src/build.ts", src)
	if err != nil {
		t.Fatalf("ExtractSymbols() error = %v", err)
	}

	var matches []Symbol
	for _, symbol := range symbols {
		if symbol.Name == "buildUser" {
			matches = append(matches, symbol)
		}
	}
	if len(matches) != 1 {
		t.Fatalf("buildUser symbols = %+v, want one top-level function symbol", matches)
	}
	if matches[0].Kind != "function" {
		t.Fatalf("buildUser kind = %q, want function", matches[0].Kind)
	}
}
