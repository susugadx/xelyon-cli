package jsast

import "testing"

func TestNamedImportAliasesWithParsed(t *testing.T) {
	src := []byte("import { Button as PrimaryButton, Link, type Button as TypeButton } from './ui'\n" +
		"import type { Button as TypeOnlyButton } from './types'\n" +
		"import DefaultButton from './default'\n" +
		"import * as UI from './ui'\n")
	parsed, err := ParseBytes("src/App.tsx", src)
	if err != nil {
		t.Fatalf("ParseBytes() error = %v", err)
	}
	defer parsed.Close()

	aliases := NamedImportAliasesWithParsed(parsed, "Button")

	if len(aliases) != 1 {
		t.Fatalf("aliases = %+v, want one value alias", aliases)
	}
	if aliases[0].Imported != "Button" || aliases[0].Local != "PrimaryButton" || aliases[0].Source != "./ui" || aliases[0].Line != 1 {
		t.Fatalf("alias = %+v, want Button as PrimaryButton on line 1", aliases[0])
	}
}

func TestNamedImportAliasesWithParsedSkipsTypeOnlyImportVariants(t *testing.T) {
	tests := []struct {
		name string
		src  string
	}{
		{
			name: "specifier keyword",
			src:  "import { type Button as PrimaryButton } from './ui'\n",
		},
		{
			name: "specifier keyword with comment",
			src:  "import { /*c*/ type Button as PrimaryButton } from './ui'\n",
		},
		{
			name: "specifier typeof keyword",
			src:  "import { typeof Button as PrimaryButton } from './ui'\n",
		},
		{
			name: "statement keyword",
			src:  "import type { Button as PrimaryButton } from './ui'\n",
		},
		{
			name: "statement keyword without space before brace",
			src:  "import type{ Button as PrimaryButton } from './ui'\n",
		},
		{
			name: "statement keyword with comment",
			src:  "import /*c*/ type { Button as PrimaryButton } from './ui'\n",
		},
		{
			name: "statement typeof keyword",
			src:  "import typeof { Button as PrimaryButton } from './ui'\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsed, err := ParseBytes("src/App.tsx", []byte(tt.src))
			if err != nil {
				t.Fatalf("ParseBytes() error = %v", err)
			}
			defer parsed.Close()

			if aliases := NamedImportAliasesWithParsed(parsed, "Button"); len(aliases) != 0 {
				t.Fatalf("aliases = %+v, want no value alias for type-only import", aliases)
			}
		})
	}
}

func TestNamedImportAliasesWithParsedKeepsValueAliasNamedType(t *testing.T) {
	src := []byte("import { type as TypeAlias, Button as PrimaryButton } from './ui'\n")
	parsed, err := ParseBytes("src/App.tsx", src)
	if err != nil {
		t.Fatalf("ParseBytes() error = %v", err)
	}
	defer parsed.Close()

	aliases := NamedImportAliasesWithParsed(parsed, "type")

	if len(aliases) != 1 {
		t.Fatalf("aliases = %+v, want value alias for imported name type", aliases)
	}
	if aliases[0].Imported != "type" || aliases[0].Local != "TypeAlias" || aliases[0].Source != "./ui" {
		t.Fatalf("alias = %+v, want type as TypeAlias", aliases[0])
	}
}

func TestJSXLocalNameUsagesWithParsed(t *testing.T) {
	src := []byte("export function App() {\n" +
		"  return <>\n" +
		"    <PrimaryButton />\n" +
		"    <PrimaryButton label=\"Save\" />\n" +
		"    <PrimaryButton>Save</PrimaryButton>\n" +
		"    <UI.PrimaryButton />\n" +
		"    <PrimaryButton.Sub />\n" +
		"    <primaryButton />\n" +
		"  </>\n" +
		"}\n" +
		"const html = \"<PrimaryButton />\"\n")
	parsed, err := ParseBytes("src/App.tsx", src)
	if err != nil {
		t.Fatalf("ParseBytes() error = %v", err)
	}
	defer parsed.Close()

	usages := JSXLocalNameUsagesWithParsed(parsed, "PrimaryButton")

	if len(usages) != 3 {
		t.Fatalf("usages = %+v, want three bare component opening usages", usages)
	}
	wantLines := []int{3, 4, 5}
	for i, wantLine := range wantLines {
		if usages[i].Line != wantLine {
			t.Fatalf("usage %d line = %d, want %d; usages=%+v", i, usages[i].Line, wantLine, usages)
		}
	}
}

func TestVisitJSXLocalNameUsagesForNamedImportAliasWithParsedStopsWhenVisitorStops(t *testing.T) {
	src := []byte("import { Button as PrimaryButton } from './Button'\n" +
		"export function App() {\n" +
		"  return <>\n" +
		"    <PrimaryButton />\n" +
		"    <PrimaryButton />\n" +
		"    <PrimaryButton />\n" +
		"    <PrimaryButton />\n" +
		"  </>\n" +
		"}\n")
	parsed, err := ParseBytes("src/App.tsx", src)
	if err != nil {
		t.Fatalf("ParseBytes() error = %v", err)
	}
	defer parsed.Close()

	aliases := NamedImportAliasesWithParsed(parsed, "Button")
	if len(aliases) != 1 {
		t.Fatalf("aliases = %+v, want one value alias", aliases)
	}
	var lines []int
	VisitJSXLocalNameUsagesForNamedImportAliasWithParsed(parsed, aliases[0], func(usage JSXLocalNameUsage) bool {
		lines = append(lines, usage.Line)
		return len(lines) < 2
	})

	if len(lines) != 2 {
		t.Fatalf("visited lines = %v, want visitor to stop after two usages", lines)
	}
}
