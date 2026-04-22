package repomap

import (
	"strings"
	"testing"
)

func TestBuild_EmptyDirectory(t *testing.T) {
	requireRipgrep(t)

	root := t.TempDir()
	pm := buildProjectMapForTest(t, root, 4000)
	if len(pm.Files) != 0 {
		t.Fatalf("Files length = %d, want 0", len(pm.Files))
	}
	if got := pm.Generate(); got != "" {
		t.Fatalf("Generate() = %q, want empty string", got)
	}
}

func TestBuild_IgnoreDirs(t *testing.T) {
	requireRipgrep(t)

	root := t.TempDir()
	writeProjectMapTestFile(t, root, "src/keep.go", "package main\nfunc Keep() {}\n")
	writeProjectMapTestFile(t, root, "node_modules/pkg/index.js", "export function ignoreMe() {}\n")
	writeProjectMapTestFile(t, root, "vendor/lib/helper.go", "package lib\nfunc IgnoreMe() {}\n")
	writeProjectMapTestFile(t, root, "generated/file.go", "package main\nfunc SkipMe() {}\n")

	pm := buildProjectMapForTest(t, root, 4000, "generated")
	output := pm.Generate()
	if !strings.Contains(output, "keep.go") {
		t.Fatalf("expected keep.go in output:\n%s", output)
	}
	for _, unwanted := range []string{"node_modules", "vendor", "generated"} {
		if strings.Contains(output, unwanted) {
			t.Fatalf("unexpected ignored directory %q in output:\n%s", unwanted, output)
		}
	}
}
