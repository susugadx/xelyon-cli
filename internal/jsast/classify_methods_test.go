package jsast

import (
	"testing"

	codeast "github.com/susugadx/xelyon-cli/internal/ast"
)

func TestClassifyLineMethods(t *testing.T) {
	tests := []struct {
		name   string
		path   string
		src    string
		line   int
		symbol string
		want   codeast.MatchClass
	}{
		{
			name: "class method definition",
			path: "src/view.ts",
			src: "class View {\n" +
				"  render() { return 'ok' }\n" +
				"}\n" +
				"new View().render()\n",
			line:   2,
			symbol: "render",
			want:   codeast.ClassDef,
		},
		{
			name: "class method caller",
			path: "src/view.ts",
			src: "class View {\n" +
				"  render() { return 'ok' }\n" +
				"}\n" +
				"new View().render()\n",
			line:   4,
			symbol: "render",
			want:   codeast.ClassCall,
		},
		{
			name: "method read before bind is not caller",
			path: "src/view.ts",
			src: "class View {\n" +
				"  render() { return 'ok' }\n" +
				"}\n" +
				"const bound = new View().render.bind(new View())\n",
			line:   4,
			symbol: "render",
			want:   codeast.ClassRef,
		},
		{
			name: "method call via call",
			path: "src/view.ts",
			src: "class View {\n" +
				"  render() { return 'ok' }\n" +
				"}\n" +
				"new View().render.call(new View())\n",
			line:   4,
			symbol: "render",
			want:   codeast.ClassCall,
		},
		{
			name: "method call via apply",
			path: "src/view.ts",
			src: "class View {\n" +
				"  render() { return 'ok' }\n" +
				"}\n" +
				"new View().render.apply(new View(), [])\n",
			line:   4,
			symbol: "render",
			want:   codeast.ClassCall,
		},
		{
			name: "method literally named call",
			path: "src/view.ts",
			src: "class View {\n" +
				"  call() { return 'ok' }\n" +
				"}\n" +
				"new View().call()\n",
			line:   4,
			symbol: "call",
			want:   codeast.ClassCall,
		},
		{
			name: "method literally named apply",
			path: "src/view.ts",
			src: "class View {\n" +
				"  apply() { return 'ok' }\n" +
				"}\n" +
				"new View().apply()\n",
			line:   4,
			symbol: "apply",
			want:   codeast.ClassCall,
		},
		{
			name: "method read before chained call is not caller",
			path: "src/view.ts",
			src: "class View {\n" +
				"  render() { return 'ok' }\n" +
				"}\n" +
				"new View().render.extra()\n",
			line:   4,
			symbol: "render",
			want:   codeast.ClassRef,
		},
		{
			name: "interface method signature",
			path: "src/store.ts",
			src: "interface Store {\n" +
				"  save(id: string): void\n" +
				"}\n",
			line:   2,
			symbol: "save",
			want:   codeast.ClassDef,
		},
		{
			name: "type alias method signature",
			path: "src/store.ts",
			src: "type Store = {\n" +
				"  save(id: string): void\n" +
				"}\n",
			line:   2,
			symbol: "save",
			want:   codeast.ClassDef,
		},
		{
			name: "intersection type alias method signature",
			path: "src/store.ts",
			src: "type Store = Base & {\n" +
				"  save(id: string): void\n" +
				"}\n",
			line:   2,
			symbol: "save",
			want:   codeast.ClassDef,
		},
		{
			name: "parenthesized type alias method signature",
			path: "src/store.ts",
			src: "type Store = ({\n" +
				"  save(id: string): void\n" +
				"})\n",
			line:   2,
			symbol: "save",
			want:   codeast.ClassDef,
		},
		{
			name: "abstract method signature",
			path: "src/base.ts",
			src: "abstract class Base {\n" +
				"  abstract run(): void\n" +
				"}\n",
			line:   2,
			symbol: "run",
			want:   codeast.ClassDef,
		},
		{
			name: "private property method definition",
			path: "src/store.ts",
			src: "class Store {\n" +
				"  #secret() { return null }\n" +
				"}\n",
			line:   2,
			symbol: "#secret",
			want:   codeast.ClassDef,
		},
		{
			name: "object literal method is not definition",
			path: "src/handlers.ts",
			src: "const handlers = {\n" +
				"  buildUser() { return 'ok' }\n" +
				"}\n",
			line:   2,
			symbol: "buildUser",
			want:   codeast.ClassRef,
		},
		{
			name: "inline object type method is not definition",
			path: "src/handlers.ts",
			src: "function useStore(store: {\n" +
				"  buildUser(id: string): void\n" +
				"}) {}\n",
			line:   2,
			symbol: "buildUser",
			want:   ClassTypeRef,
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
