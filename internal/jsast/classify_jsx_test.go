package jsast

import (
	"testing"

	codeast "github.com/susugadx/xelyon-cli/internal/ast"
)

func TestClassifyLineJSXTags(t *testing.T) {
	src := []byte("export function App() {\n" +
		"  return <>\n" +
		"    <Button>\n" +
		"      save\n" +
		"    </Button>\n" +
		"    <buildUser />\n" +
		"  </>\n" +
		"}\n")

	tests := []struct {
		name   string
		line   int
		symbol string
		want   codeast.MatchClass
	}{
		{name: "component opening", line: 3, symbol: "Button", want: codeast.ClassCall},
		{name: "component closing", line: 5, symbol: "Button", want: ClassIgnored},
		{name: "lowercase intrinsic", line: 6, symbol: "buildUser", want: ClassIgnored},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ClassifyLine("src/App.tsx", src, tt.line, tt.symbol)
			if err != nil {
				t.Fatalf("ClassifyLine() error = %v", err)
			}
			if got.Class != tt.want {
				t.Fatalf("ClassifyLine() class = %s, want %s", got.Class, tt.want)
			}
		})
	}
}
