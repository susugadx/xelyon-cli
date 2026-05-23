package jsast

import "testing"

func TestExportBindingsWithParsedCollectsReExportSources(t *testing.T) {
	src := []byte("export { Button, Link as Anchor } from './ui'\n" +
		"export { LocalButton as Button }\n" +
		"export default function App() { return null }\n")
	parsed, err := ParseBytes("src/index.tsx", src)
	if err != nil {
		t.Fatalf("ParseBytes() error = %v", err)
	}
	defer parsed.Close()

	bindings := ExportBindingsWithParsed(parsed)

	want := []ExportBinding{
		{Local: "Button", Exported: "Button", Source: "./ui", Line: 1},
		{Local: "Link", Exported: "Anchor", Source: "./ui", Line: 1},
		{Local: "LocalButton", Exported: "Button", Line: 2},
	}
	if len(bindings) != len(want) {
		t.Fatalf("bindings = %+v, want %+v", bindings, want)
	}
	for i := range want {
		if bindings[i] != want[i] {
			t.Fatalf("binding %d = %+v, want %+v", i, bindings[i], want[i])
		}
	}
}

func TestSymbolExportedAsDefaultWithParsed(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{
			name: "export specifier default alias",
			src:  "function buildUser() {}\nexport { buildUser as default }\n",
		},
		{
			name: "export default identifier",
			src:  "function buildUser() {}\nexport default buildUser\n",
		},
		{
			name: "export default declaration",
			src:  "export default function buildUser() {}\n",
		},
		{
			name: "commonjs default identifier",
			src:  "function buildUser() {}\nmodule.exports = buildUser\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := ParseBytes("src/build.js", []byte(tt.src))
			if err != nil {
				t.Fatalf("ParseBytes() error = %v", err)
			}
			defer parsed.Close()

			if !SymbolExportedAsDefaultWithParsed(parsed, "buildUser") {
				t.Fatalf("SymbolExportedAsDefaultWithParsed(buildUser) = false")
			}
			if SymbolExportedAsDefaultWithParsed(parsed, "otherUser") {
				t.Fatalf("SymbolExportedAsDefaultWithParsed(otherUser) = true")
			}
		})
	}
}
