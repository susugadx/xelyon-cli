package jsast

import (
	"testing"

	codeast "github.com/susugadx/xelyon-cli/internal/ast"
)

func TestClassifyLine(t *testing.T) {
	src := []byte("function buildUser(id) { return id }\n" +
		"buildUser('1')\n" +
		"const raw = `buildUser()`\n" +
		"const label = `${buildUser()}`\n" +
		"/*\n" +
		"buildUser()\n" +
		"*/\n" +
		"const text = \"buildUser()\"; buildUser()\n" +
		"export { buildUser }\n" +
		"exports[\"buildUser\"] = buildUser\n" +
		"const { buildUser: requiredBuildUser } = require('./build')\n" +
		"const quoted = `${\"buildUser()\"}`\n" +
		"const buildUser = \"require\"\n" +
		"exports[\"notbuildUser\"] = buildUser\n" +
		"export { buildUser as createUser }\n" +
		"export default buildUser\n" +
		"export { other as buildUser }\n" +
		"const nested = `${items.map(() => `${buildUser()}`)}`\n" +
		"const nestedRaw = `${`buildUser()`}`\n" +
		"const re = /[//]/; buildUser('regex')\n" +
		"const regexOnly = /buildUser/\n")

	tests := []struct {
		line int
		want codeast.MatchClass
	}{
		{1, codeast.ClassDef},
		{2, codeast.ClassCall},
		{3, codeast.ClassString},
		{4, codeast.ClassCall},
		{6, codeast.ClassComment},
		{8, codeast.ClassCall},
		{9, ClassExport},
		{10, ClassExport},
		{11, codeast.ClassImport},
		{12, codeast.ClassString},
		{13, codeast.ClassDef},
		{14, ClassExport},
		{15, ClassExport},
		{16, ClassExport},
		{17, ClassIgnored},
		{18, codeast.ClassCall},
		{19, codeast.ClassString},
		{20, codeast.ClassCall},
		{21, codeast.ClassString},
	}

	for _, tt := range tests {
		got, err := ClassifyLine("src/build.js", src, tt.line, "buildUser")
		if err != nil {
			t.Fatalf("ClassifyLine(line %d) error = %v", tt.line, err)
		}
		if got.Class != tt.want {
			t.Fatalf("ClassifyLine(line %d) = %s, want %s", tt.line, got.Class, tt.want)
		}
	}
}

func TestClassifyRangeUsesLSPLocationText(t *testing.T) {
	importLine := "import { buildUser as createUser } from './build'"
	callLine := "createUser()"
	exportLine := "exports.createUser = buildUser"
	src := []byte(importLine + "\n" + callLine + "\n" + exportLine + "\n")
	parsed, err := ParseBytes("src/app.js", src)
	if err != nil {
		t.Fatalf("ParseBytes() error = %v", err)
	}
	defer parsed.Close()

	start, end := testLSPRangeForToken(callLine, "createUser")
	got, err := ClassifyRangeWithParsed(parsed, 2, start, 2, end, "buildUser")
	if err != nil {
		t.Fatalf("ClassifyRangeWithParsed(call) error = %v", err)
	}
	if got.Class != codeast.ClassCall {
		t.Fatalf("alias call class = %s, want call", got.Class)
	}

	start, end = testLSPRangeForToken(importLine, "createUser")
	got, err = ClassifyRangeWithParsed(parsed, 1, start, 1, end, "buildUser")
	if err != nil {
		t.Fatalf("ClassifyRangeWithParsed(import) error = %v", err)
	}
	if got.Class != codeast.ClassImport {
		t.Fatalf("alias import class = %s, want import", got.Class)
	}

	start, end = testLSPRangeForToken(exportLine, "buildUser")
	got, err = ClassifyRangeWithParsed(parsed, 3, start, 3, end, "buildUser")
	if err != nil {
		t.Fatalf("ClassifyRangeWithParsed(export) error = %v", err)
	}
	if got.Class != ClassExport {
		t.Fatalf("commonjs alias export class = %s, want export", got.Class)
	}
}

func TestClassifyLineTSX(t *testing.T) {
	src := []byte("export function App() { return <Button /> }\n" +
		"export function Namespaced() { return <UI.Button /> }\n" +
		"export function Sub() { return <Button.Sub /> }\n" +
		"const html = \"<Button />\"\n" +
		"const props: ButtonProps = { label: 'Save' }\n" +
		"export function Children() { return <Button>Save</Button> }\n")

	got, err := ClassifyLine("src/App.tsx", src, 1, "Button")
	if err != nil {
		t.Fatalf("ClassifyLine() error = %v", err)
	}
	if got.Class != codeast.ClassCall {
		t.Fatalf("Button JSX class = %s, want call", got.Class)
	}
	got, err = ClassifyLine("src/App.tsx", src, 6, "Button")
	if err != nil {
		t.Fatalf("ClassifyLine(children) error = %v", err)
	}
	if got.Class != codeast.ClassCall {
		t.Fatalf("Button JSX children class = %s, want call", got.Class)
	}
	for _, line := range []int{2, 3, 4} {
		got, err := ClassifyLine("src/App.tsx", src, line, "Button")
		if err != nil {
			t.Fatalf("ClassifyLine(line %d) error = %v", line, err)
		}
		if got.Class == codeast.ClassCall {
			t.Fatalf("ClassifyLine(line %d) = call, want non-call", line)
		}
	}
	got, err = ClassifyLine("src/App.tsx", src, 5, "ButtonProps")
	if err != nil {
		t.Fatalf("ClassifyLine(type ref) error = %v", err)
	}
	if got.Class != ClassTypeRef {
		t.Fatalf("ButtonProps class = %s, want type_ref", got.Class)
	}
}

func TestClassifyLineAbstractClassDefinitionAndScope(t *testing.T) {
	src := []byte("abstract class AbstractBuilder {\n" +
		"  static create() { return new AbstractBuilder() }\n" +
		"}\n")

	got, err := ClassifyLine("src/base.ts", src, 1, "AbstractBuilder")
	if err != nil {
		t.Fatalf("ClassifyLine(definition) error = %v", err)
	}
	if got.Class != codeast.ClassDef {
		t.Fatalf("definition class = %s, want def", got.Class)
	}
	if got.Scope != "class AbstractBuilder" {
		t.Fatalf("definition scope = %q, want class AbstractBuilder", got.Scope)
	}
}
