package jsast

import (
	"testing"

	codeast "github.com/susugadx/xelyon-cli/internal/ast"
)

func TestClassifyLineProperties(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		src    string
		line   int
		symbol string
		want   codeast.MatchClass
	}{
		{
			name: "class field definition",
			path: "src/view.ts",
			src: "class View {\n" +
				"  render = () => 'ok'\n" +
				"}\n" +
				"new View().render()\n",
			line:   2,
			symbol: "render",
			want:   codeast.ClassDef,
		},
		{
			name: "class field caller",
			path: "src/view.ts",
			src: "class View {\n" +
				"  render = () => 'ok'\n" +
				"}\n" +
				"new View().render()\n",
			line:   4,
			symbol: "render",
			want:   codeast.ClassCall,
		},
		{
			name: "interface property signature",
			path: "src/store.ts",
			src: "interface Store {\n" +
				"  save: (id: string) => void\n" +
				"}\n",
			line:   2,
			symbol: "save",
			want:   codeast.ClassDef,
		},
		{
			name: "type alias property signature",
			path: "src/store.ts",
			src: "type Store = {\n" +
				"  save: (id: string) => void\n" +
				"}\n",
			line:   2,
			symbol: "save",
			want:   codeast.ClassDef,
		},
		{
			name: "intersection type alias property signature",
			path: "src/store.ts",
			src: "type Store = Base & {\n" +
				"  save: (id: string) => void\n" +
				"}\n",
			line:   2,
			symbol: "save",
			want:   codeast.ClassDef,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ClassifyLine(tt.path, []byte(tt.src), tt.line, tt.symbol)
			if err != nil {
				t.Fatalf("ClassifyLine() error = %v", err)
			}
			if got.Class != tt.want {
				t.Fatalf("ClassifyLine() class = %s, want %s", got.Class, tt.want)
			}
		})
	}
}

func TestClassifyLinePropertiesDoNotTreatObjectShapeAsDefinition(t *testing.T) {
	tests := []struct {
		name string
		path string
		src  string
		line int
	}{
		{
			name: "object literal property",
			path: "src/handlers.ts",
			src: "const handlers = {\n" +
				"  buildUser: () => 'ok'\n" +
				"}\n",
			line: 2,
		},
		{
			name: "inline object type property",
			path: "src/handlers.ts",
			src: "function useStore(store: {\n" +
				"  buildUser: (id: string) => void\n" +
				"}) {}\n",
			line: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ClassifyLine(tt.path, []byte(tt.src), tt.line, "buildUser")
			if err != nil {
				t.Fatalf("ClassifyLine() error = %v", err)
			}
			if got.Class == codeast.ClassDef {
				t.Fatalf("ClassifyLine() class = def, want non-def")
			}
		})
	}
}
