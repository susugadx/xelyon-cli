package repomap

import (
	"strings"
	"testing"
)

func TestBuild_GoFileUsesAST(t *testing.T) {
	requireRipgrep(t)

	root := t.TempDir()
	writeProjectMapTestFile(t, root, "internal/agent/agent.go", "package agent\n\ntype Agent struct{}\n\nfunc (a *Agent) maybeAutoCompress(\n\tctx context.Context,\n) bool {\n\t_ = ctx\n\treturn true\n}\n\ntype Config = map[string]string\n")

	pm := buildProjectMapForTest(t, root, 4000)
	file := findFileEntry(t, pm, "internal/agent/agent.go")

	method := findSymbol(t, file, "maybeAutoCompress")
	if method.Kind != "method" {
		t.Fatalf("method kind = %q, want method", method.Kind)
	}
	if method.Line != 5 || method.EndLine != 10 {
		t.Fatalf("method location = %d-%d, want 5-10", method.Line, method.EndLine)
	}
	if method.Exported {
		t.Fatal("method should not be exported")
	}
	if !strings.Contains(method.Signature, "func (a *Agent) maybeAutoCompress(") {
		t.Fatalf("method signature = %q", method.Signature)
	}

	alias := findSymbol(t, file, "Config")
	if alias.Kind != "type" {
		t.Fatalf("alias kind = %q, want type", alias.Kind)
	}
}

func TestBuild_GoProject(t *testing.T) {
	requireRipgrep(t)

	root := t.TempDir()
	writeProjectMapTestFile(t, root, "main.go", "package main\n\nconst version = \"v1\"\n\ntype Builder struct {}\n\nfunc Build() error {\n\treturn nil\n}\n")

	pm := buildProjectMapForTest(t, root, 4000)
	file := findFileEntry(t, pm, "main.go")
	got := strings.Join(signatures(file), "\n")
	for _, want := range []string{"const version = \"v1\"", "type Builder struct", "func Build() error"} {
		if !strings.Contains(got, want) {
			t.Fatalf("Go signatures missing %q from:\n%s", want, got)
		}
	}
}
