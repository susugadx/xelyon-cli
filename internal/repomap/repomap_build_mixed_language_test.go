package repomap

import "testing"

func TestBuild_MixedProject(t *testing.T) {
	requireRipgrep(t)

	root := t.TempDir()
	writeProjectMapTestFile(t, root, "main.go", "package main\n\nfunc Build(\n\tctx context.Context,\n) error {\n\t_ = ctx\n\treturn nil\n}\n")
	writeProjectMapTestFile(t, root, "pkg/tasks.py", "class TaskRunner:\n    pass\n")
	writeProjectMapTestFile(t, root, "src/app.ts", "export function buildMap() {}\n")

	pm := buildProjectMapForTest(t, root, 4000)

	goFile := findFileEntry(t, pm, "main.go")
	goSymbol := findSymbol(t, goFile, "Build")
	if goSymbol.Kind != "function" {
		t.Fatalf("Go kind = %q, want function", goSymbol.Kind)
	}
	if goSymbol.EndLine != 8 {
		t.Fatalf("Go EndLine = %d, want 8", goSymbol.EndLine)
	}

	pyFile := findFileEntry(t, pm, "pkg/tasks.py")
	pySymbol := findSymbol(t, pyFile, "TaskRunner")
	if pySymbol.Kind != "class" {
		t.Fatalf("Python kind = %q, want class", pySymbol.Kind)
	}

	tsFile := findFileEntry(t, pm, "src/app.ts")
	tsSymbol := findSymbol(t, tsFile, "buildMap")
	if tsSymbol.Kind != "function" {
		t.Fatalf("TypeScript kind = %q, want function", tsSymbol.Kind)
	}
}
