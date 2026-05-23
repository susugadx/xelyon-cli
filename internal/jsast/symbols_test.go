package jsast

import (
	"slices"
	"testing"
)

func TestExtractSymbols(t *testing.T) {
	src := []byte(`
export function buildUser(id) { return id }
export const makeUser = async (id) => buildUser(id)
class UserBuilder {}
module.exports = function buildOrg() { return "org" }
exports.buildTeam = function(id) { return id }
type BuildOptions = { id: string }
interface User { id: string }
`)

	symbols, err := ExtractSymbols("src/build.ts", src)
	if err != nil {
		t.Fatalf("ExtractSymbols() error = %v", err)
	}

	for _, want := range []struct {
		name string
		kind string
	}{
		{"buildUser", "function"},
		{"makeUser", "function"},
		{"UserBuilder", "class"},
		{"buildOrg", "function"},
		{"buildTeam", "function"},
		{"BuildOptions", "type"},
		{"User", "interface"},
	} {
		if !slices.ContainsFunc(symbols, func(symbol Symbol) bool {
			return symbol.Name == want.name && symbol.Kind == want.kind
		}) {
			t.Fatalf("symbols = %+v, want %s %s", symbols, want.kind, want.name)
		}
	}
}

func TestExtractSymbolsUsesLSPCharacterOffsets(t *testing.T) {
	line := "const marker = \"😀\"; function buildUser() {}"
	symbols, err := ExtractSymbols("src/build.js", []byte(line+"\n"))
	if err != nil {
		t.Fatalf("ExtractSymbols() error = %v", err)
	}
	wantStart, _ := testLSPRangeForToken(line, "buildUser")
	if !slices.ContainsFunc(symbols, func(symbol Symbol) bool {
		return symbol.Name == "buildUser" && symbol.Character == wantStart
	}) {
		t.Fatalf("symbols = %+v, want buildUser character %d", symbols, wantStart)
	}
}

func TestExtractSymbolsAbstractClass(t *testing.T) {
	symbols, err := ExtractSymbols("src/base.ts", []byte("abstract class AbstractBuilder {}\n"))
	if err != nil {
		t.Fatalf("ExtractSymbols() error = %v", err)
	}
	if !slices.ContainsFunc(symbols, func(symbol Symbol) bool {
		return symbol.Name == "AbstractBuilder" && symbol.Kind == "class" && !symbol.Exported
	}) {
		t.Fatalf("symbols = %+v, want non-exported class AbstractBuilder", symbols)
	}
}

func TestExtractSymbolsIgnoresFallbackDefinitionsInsideCommentsAndTemplates(t *testing.T) {
	src := []byte("/*\n" +
		"export function commentedBuildUser() { return 'comment' }\n" +
		"*/\n" +
		"const raw = `\n" +
		"export function templatedBuildUser() { return 'template' }\n" +
		"`\n" +
		"export function realBuildUser() { return 'real' }\n")

	symbols, err := ExtractSymbols("src/build.ts", src)
	if err != nil {
		t.Fatalf("ExtractSymbols() error = %v", err)
	}

	for _, name := range []string{"commentedBuildUser", "templatedBuildUser"} {
		if slices.ContainsFunc(symbols, func(symbol Symbol) bool {
			return symbol.Name == name
		}) {
			t.Fatalf("symbols = %+v, did not want fallback definition %s from non-code text", symbols, name)
		}
	}
	if !slices.ContainsFunc(symbols, func(symbol Symbol) bool {
		return symbol.Name == "realBuildUser" && symbol.Kind == "function"
	}) {
		t.Fatalf("symbols = %+v, want realBuildUser", symbols)
	}
}
