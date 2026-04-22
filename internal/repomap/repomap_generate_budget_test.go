package repomap

import (
	"strconv"
	"strings"
	"testing"
)

func TestGenerate_TokenLimit(t *testing.T) {
	requireRipgrep(t)

	root := t.TempDir()
	writeProjectMapTestFile(t, root, "pkg/service.go", "package pkg\n\nfunc BuildOne() {}\nfunc BuildTwo() {}\n")

	var testFile strings.Builder
	testFile.WriteString("package pkg\n\n")
	for i := 0; i < 25; i++ {
		testFile.WriteString("func TestServiceCase")
		testFile.WriteString(strconv.Itoa(i))
		testFile.WriteString("() {}\n")
	}
	writeProjectMapTestFile(t, root, "pkg/service_test.go", testFile.String())

	pm := buildProjectMapForTest(t, root, 80)
	output := pm.Generate()
	if !strings.Contains(output, "service_test.go") {
		t.Fatalf("expected test file to remain in output:\n%s", output)
	}
	if strings.Contains(output, "func TestServiceCase0()") {
		t.Fatalf("expected test symbols to be omitted under token limit:\n%s", output)
	}
	if !strings.Contains(output, "func BuildOne()") {
		t.Fatalf("expected implementation symbol to remain:\n%s", output)
	}
}

func TestGenerate_HandlesNilEmptyAndBudgetReduction(t *testing.T) {
	if got := (*ProjectMap)(nil).Generate(); got != "" {
		t.Fatalf("nil Generate() = %q, want empty string", got)
	}

	empty := &ProjectMap{}
	if got := empty.Generate(); got != "" {
		t.Fatalf("empty Generate() = %q, want empty string", got)
	}

	pm := &ProjectMap{
		MaxTokens: 18,
		Files: []*FileEntry{
			{Path: "pkg/service.go", LineCount: 40, Symbols: []Symbol{{Name: "Build", Line: 10, Signature: "func Build()"}}},
			{Path: "pkg/service_test.go", LineCount: 20, Symbols: []Symbol{{Name: "TestBuild", Line: 5, Signature: "func TestBuild()"}}},
			{Path: "internal/deep/nested.go", LineCount: 10, Symbols: []Symbol{{Name: "Nested", Line: 3, Signature: "func Nested()"}}},
		},
	}

	output := pm.Generate()
	if output == "" {
		t.Fatal("Generate() returned empty output unexpectedly")
	}
	if !strings.Contains(output, "truncated to fit token budget") {
		t.Fatalf("Generate() should report truncation under small budget, got:\n%s", output)
	}
	if strings.Contains(output, "func TestBuild()") {
		t.Fatalf("Generate() should drop test symbols before omitting files, got:\n%s", output)
	}
}
