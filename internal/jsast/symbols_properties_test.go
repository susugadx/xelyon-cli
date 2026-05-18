package jsast

import (
	"slices"
	"testing"
)

func TestExtractSymbolsProperties(t *testing.T) {
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
			name: "class fields",
			path: "src/service.ts",
			src: `
class UserService {
  buildUser = (id: string) => id
  count = 0
}
`,
			want: []struct {
				name string
				kind string
			}{
				{"buildUser", "field"},
				{"count", "field"},
			},
		},
		{
			name: "interface property signatures",
			path: "src/store.ts",
			src: `
interface UserStore {
  saveUser: (id: string) => void
  count: number
}
`,
			want: []struct {
				name string
				kind string
			}{
				{"saveUser", "property"},
				{"count", "property"},
			},
		},
		{
			name: "intersection type alias property signature",
			path: "src/store.ts",
			src: `
type BaseStore = { id: string }
type UserStore = BaseStore & {
  saveUser: (id: string) => void
}
`,
			want: []struct {
				name string
				kind string
			}{
				{"saveUser", "property"},
			},
		},
		{
			name: "parenthesized type alias property signature",
			path: "src/store.ts",
			src: `
type UserStore = ({
  saveUser: (id: string) => void
})
`,
			want: []struct {
				name string
				kind string
			}{
				{"saveUser", "property"},
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

func TestExtractSymbolsPrivateAndProtectedFieldsAreNotExported(t *testing.T) {
	src := []byte(`
export class UserStore {
  public saveUser = (id: string) => id
  private buildCache = () => null
  protected resetCache: () => void
  #secret = null
}
`)

	symbols, err := ExtractSymbols("src/store.ts", src)
	if err != nil {
		t.Fatalf("ExtractSymbols() error = %v", err)
	}

	assertFieldExported := func(name string, want bool) {
		t.Helper()
		for _, symbol := range symbols {
			if symbol.Name == name && symbol.Kind == "field" {
				if symbol.Exported != want {
					t.Fatalf("%s Exported = %v, want %v; symbols = %+v", name, symbol.Exported, want, symbols)
				}
				return
			}
		}
		t.Fatalf("symbols = %+v, want field %s", symbols, name)
	}
	assertFieldExported("saveUser", true)
	assertFieldExported("buildCache", false)
	assertFieldExported("resetCache", false)
	assertFieldExported("#secret", false)
}

func TestExtractSymbolsSkipsInlineObjectTypePropertiesAsTopLevelSymbols(t *testing.T) {
	src := []byte(`
export function buildUser(id: string) { return id }
function useStore(store: {
  buildUser: (id: string) => void
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

func TestExtractSymbolsSkipsObjectLiteralPropertiesAsTopLevelSymbols(t *testing.T) {
	src := []byte(`
export function buildUser(id: string) { return id }
export const handlers = {
  buildUser: (id: string) => id
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
