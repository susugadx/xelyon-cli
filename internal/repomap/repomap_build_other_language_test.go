package repomap

import (
	"strings"
	"testing"
)

func TestBuild_NonGoFileUsesRegex(t *testing.T) {
	requireRipgrep(t)

	root := t.TempDir()
	writeProjectMapTestFile(t, root, "pkg/tasks.py", "class Builder:\n    pass\n\nasync def build_map():\n    return True\n")

	pm := buildProjectMapForTest(t, root, 4000)
	file := findFileEntry(t, pm, "pkg/tasks.py")

	classSymbol := findSymbol(t, file, "Builder")
	if classSymbol.Kind != "class" {
		t.Fatalf("class kind = %q, want class", classSymbol.Kind)
	}
	if classSymbol.EndLine != 0 {
		t.Fatalf("class EndLine = %d, want 0", classSymbol.EndLine)
	}

	funcSymbol := findSymbol(t, file, "build_map")
	if funcSymbol.Kind != "function" {
		t.Fatalf("function kind = %q, want function", funcSymbol.Kind)
	}
	if funcSymbol.Signature != "async def build_map():" {
		t.Fatalf("function signature = %q", funcSymbol.Signature)
	}
}

func TestBuild_TypeScriptProject(t *testing.T) {
	requireRipgrep(t)

	root := t.TempDir()
	writeProjectMapTestFile(t, root, "src/app.ts", "export interface Config {}\nexport class Builder {}\nexport function buildMap() {}\nexport  const buildUser = <T>(value: T): T => value\nexport\tfunction buildTeam() {}\nexport default function buildOrg() {}\nexport default class UserBuilder {}\n")
	writeProjectMapTestFile(t, root, "src/types.d.ts", "export declare function buildDeclared(): string;\ndeclare class DeclaredBuilder {}\n")

	pm := buildProjectMapForTest(t, root, 4000)
	file := findFileEntry(t, pm, "src/app.ts")
	got := strings.Join(signatures(file), "\n")
	for _, want := range []string{
		"export interface Config",
		"export class Builder",
		"export function buildMap()",
		"export  const buildUser = <T>(value: T): T => value",
		"export\tfunction buildTeam()",
		"export default function buildOrg()",
		"export default class UserBuilder",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("TypeScript signatures missing %q from:\n%s", want, got)
		}
	}

	declarationFile := findFileEntry(t, pm, "src/types.d.ts")
	declarationSignatures := strings.Join(signatures(declarationFile), "\n")
	for _, want := range []string{
		"export declare function buildDeclared(): string;",
		"declare class DeclaredBuilder",
	} {
		if !strings.Contains(declarationSignatures, want) {
			t.Fatalf("TypeScript declaration signatures missing %q from:\n%s", want, declarationSignatures)
		}
	}
}

func TestBuild_PythonProject(t *testing.T) {
	requireRipgrep(t)

	root := t.TempDir()
	writeProjectMapTestFile(t, root, "pkg/tasks.py", "class Builder:\n    pass\n\nasync def build_map():\n    return True\n")

	pm := buildProjectMapForTest(t, root, 4000)
	file := findFileEntry(t, pm, "pkg/tasks.py")
	got := strings.Join(signatures(file), "\n")
	for _, want := range []string{"class Builder:", "async def build_map():"} {
		if !strings.Contains(got, want) {
			t.Fatalf("Python signatures missing %q from:\n%s", want, got)
		}
	}
}

func TestBuild_RustProject(t *testing.T) {
	requireRipgrep(t)

	root := t.TempDir()
	writeProjectMapTestFile(t, root, "src/lib.rs", "pub struct Builder {}\n\npub async fn build_map() {}\n\npub trait Runnable {}\n\nimpl Builder {}\n")

	pm := buildProjectMapForTest(t, root, 4000)
	file := findFileEntry(t, pm, "src/lib.rs")
	got := strings.Join(signatures(file), "\n")
	for _, want := range []string{"pub struct Builder", "pub async fn build_map()", "pub trait Runnable", "impl Builder"} {
		if !strings.Contains(got, want) {
			t.Fatalf("Rust signatures missing %q from:\n%s", want, got)
		}
	}
}
