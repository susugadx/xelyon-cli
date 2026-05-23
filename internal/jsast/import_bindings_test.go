package jsast

import (
	"testing"

	codeast "github.com/susugadx/xelyon-cli/internal/ast"
)

func TestImportBindingsWithParsedCollectsValueBindings(t *testing.T) {
	src := []byte("import DefaultButton, { Button as PrimaryButton, Link } from './ui'\n" +
		"import type TypeButton from './types'\n" +
		"import { type Button as TypeOnlyButton } from './types'\n")
	parsed, err := ParseBytes("src/App.tsx", src)
	if err != nil {
		t.Fatalf("ParseBytes() error = %v", err)
	}
	defer parsed.Close()

	bindings := ImportBindingsWithParsed(parsed)

	want := []ImportBinding{
		{Kind: ImportBindingDefault, Imported: "default", Local: "DefaultButton", Source: "./ui", Line: 1},
		{Kind: ImportBindingNamed, Imported: "Button", Local: "PrimaryButton", Source: "./ui", Line: 1},
		{Kind: ImportBindingNamed, Imported: "Link", Local: "Link", Source: "./ui", Line: 1},
	}
	if len(bindings) != len(want) {
		t.Fatalf("bindings = %+v, want %+v", bindings, want)
	}
	for i := range want {
		if bindings[i].Kind != want[i].Kind ||
			bindings[i].Imported != want[i].Imported ||
			bindings[i].Local != want[i].Local ||
			bindings[i].Source != want[i].Source ||
			bindings[i].Line != want[i].Line {
			t.Fatalf("binding %d = %+v, want %+v", i, bindings[i], want[i])
		}
	}
}

func TestImportBindingCoversLineForMultilineImports(t *testing.T) {
	src := []byte("import {\n" +
		"  default as makeUser,\n" +
		"  buildUser as createUser,\n" +
		"} from './build'\n" +
		"import type {\n" +
		"  BuildOptions as Options,\n" +
		"} from './types'\n")
	parsed, err := ParseBytes("src/App.ts", src)
	if err != nil {
		t.Fatalf("ParseBytes() error = %v", err)
	}
	defer parsed.Close()

	valueBindings := ImportBindingsWithParsed(parsed)
	if len(valueBindings) != 2 {
		t.Fatalf("valueBindings = %+v, want two value bindings", valueBindings)
	}
	for _, binding := range valueBindings {
		for _, line := range []int{1, 2, 3, 4} {
			if !ImportBindingCoversLine(binding, line) {
				t.Fatalf("ImportBindingCoversLine(%+v, %d) = false", binding, line)
			}
		}
		if ImportBindingCoversLine(binding, 5) {
			t.Fatalf("ImportBindingCoversLine(%+v, 5) = true", binding)
		}
	}

	typeBindings := TypeImportBindingsWithParsed(parsed)
	if len(typeBindings) != 1 {
		t.Fatalf("typeBindings = %+v, want one type binding", typeBindings)
	}
	for _, line := range []int{5, 6, 7} {
		if !ImportBindingCoversLine(typeBindings[0], line) {
			t.Fatalf("ImportBindingCoversLine(%+v, %d) = false", typeBindings[0], line)
		}
	}
	if ImportBindingCoversLine(typeBindings[0], 4) {
		t.Fatalf("ImportBindingCoversLine(%+v, 4) = true", typeBindings[0])
	}
}

func TestTypeImportBindingsWithParsedCollectsTypeOnlyBindings(t *testing.T) {
	src := []byte("import DefaultButton, { Button as PrimaryButton } from './ui'\n" +
		"import type { Button as TypeButtonAlias, Link } from './types'\n" +
		"import /*c*/ type { Button as CommentButton } from './commented'\n" +
		"import type{ Button as TightButton } from './tight'\n" +
		"import { type Props as TypeProps, type Theme } from './props'\n" +
		"import { /*c*/ type Token as CommentToken } from './tokens'\n" +
		"import { type as TypeAlias } from './keywords'\n")
	parsed, err := ParseBytes("src/App.tsx", src)
	if err != nil {
		t.Fatalf("ParseBytes() error = %v", err)
	}
	defer parsed.Close()

	bindings := TypeImportBindingsWithParsed(parsed)

	want := []ImportBinding{
		{Kind: ImportBindingType, Imported: "Button", Local: "TypeButtonAlias", Source: "./types", Line: 2},
		{Kind: ImportBindingType, Imported: "Link", Local: "Link", Source: "./types", Line: 2},
		{Kind: ImportBindingType, Imported: "Button", Local: "CommentButton", Source: "./commented", Line: 3},
		{Kind: ImportBindingType, Imported: "Button", Local: "TightButton", Source: "./tight", Line: 4},
		{Kind: ImportBindingType, Imported: "Props", Local: "TypeProps", Source: "./props", Line: 5},
		{Kind: ImportBindingType, Imported: "Theme", Local: "Theme", Source: "./props", Line: 5},
		{Kind: ImportBindingType, Imported: "Token", Local: "CommentToken", Source: "./tokens", Line: 6},
	}
	if len(bindings) != len(want) {
		t.Fatalf("bindings = %+v, want %+v", bindings, want)
	}
	for i := range want {
		if bindings[i].Kind != want[i].Kind ||
			bindings[i].Imported != want[i].Imported ||
			bindings[i].Local != want[i].Local ||
			bindings[i].Source != want[i].Source ||
			bindings[i].Line != want[i].Line {
			t.Fatalf("binding %d = %+v, want %+v", i, bindings[i], want[i])
		}
	}
}

func TestImportBindingUsagesWithParsedClassifiesAliasUsages(t *testing.T) {
	src := []byte("import { buildUser as createUser } from './build'\n" +
		"const copy = createUser\n" +
		"createUser('1')\n" +
		"new createUser()\n" +
		"type Builder = typeof createUser\n" +
		"function shadowed(createUser) { return createUser() }\n")
	parsed, err := ParseBytes("src/App.ts", src)
	if err != nil {
		t.Fatalf("ParseBytes() error = %v", err)
	}
	defer parsed.Close()

	bindings := ImportBindingsWithParsed(parsed)
	if len(bindings) != 1 {
		t.Fatalf("bindings = %+v, want one value binding", bindings)
	}
	usages := ImportBindingUsagesWithParsed(parsed, bindings[0])

	want := []struct {
		line  int
		class codeast.MatchClass
	}{
		{line: 2, class: codeast.ClassRef},
		{line: 3, class: codeast.ClassCall},
		{line: 4, class: codeast.ClassCall},
		{line: 5, class: codeast.ClassRef},
	}
	if len(usages) != len(want) {
		t.Fatalf("usages = %+v, want %+v", usages, want)
	}
	for i := range want {
		if usages[i].Line != want[i].line || usages[i].Class != want[i].class {
			t.Fatalf("usage %d = %+v, want line=%d class=%s; usages=%+v", i, usages[i], want[i].line, want[i].class, usages)
		}
	}
}

func TestImportBindingUsagesWithParsedClassifiesTypeOnlyAliasUsages(t *testing.T) {
	src := []byte("import type { BuildOptions as Options } from './types'\n" +
		"const input = {} as Options\n" +
		"function render(props: Options) {}\n" +
		"class Builder implements Options {}\n")
	parsed, err := ParseBytes("src/App.ts", src)
	if err != nil {
		t.Fatalf("ParseBytes() error = %v", err)
	}
	defer parsed.Close()

	bindings := TypeImportBindingsWithParsed(parsed)
	if len(bindings) != 1 {
		t.Fatalf("bindings = %+v, want one type-only binding", bindings)
	}
	usages := ImportBindingUsagesWithParsed(parsed, bindings[0])

	want := []struct {
		line  int
		class codeast.MatchClass
	}{
		{line: 2, class: ClassTypeRef},
		{line: 3, class: ClassTypeRef},
		{line: 4, class: ClassTypeRef},
	}
	if len(usages) != len(want) {
		t.Fatalf("usages = %+v, want %+v", usages, want)
	}
	for i := range want {
		if usages[i].Line != want[i].line || usages[i].Class != want[i].class {
			t.Fatalf("usage %d = %+v, want line=%d class=%s; usages=%+v", i, usages[i], want[i].line, want[i].class, usages)
		}
	}
}

func TestImportBindingUsagesWithParsedClassifiesDefaultImportJSXUsage(t *testing.T) {
	src := []byte("import PrimaryButton from './Button'\n" +
		"export function App() { return <PrimaryButton /> }\n")
	parsed, err := ParseBytes("src/App.tsx", src)
	if err != nil {
		t.Fatalf("ParseBytes() error = %v", err)
	}
	defer parsed.Close()

	bindings := ImportBindingsWithParsed(parsed)
	if len(bindings) != 1 {
		t.Fatalf("bindings = %+v, want default import binding", bindings)
	}
	usages := ImportBindingUsagesWithParsed(parsed, bindings[0])

	if len(usages) != 1 || usages[0].Line != 2 || usages[0].Class != codeast.ClassCall {
		t.Fatalf("usages = %+v, want JSX call usage on line 2", usages)
	}
}
