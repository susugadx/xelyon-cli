package repomap

import (
	"strings"
	"testing"
)

func TestGenerate_EndLineFormat(t *testing.T) {
	pm := &ProjectMap{
		Files: []*FileEntry{
			{
				Path:      "internal/agent/agent.go",
				LineCount: 120,
				Symbols: []Symbol{
					{Line: 21, EndLine: 85, Signature: "func (a *Agent) maybeAutoCompress() bool"},
					{Line: 90, Signature: "async def build_map():"},
				},
			},
		},
	}

	output := pm.Generate()
	if !strings.Contains(output, "21-85: func (a *Agent) maybeAutoCompress() bool") {
		t.Fatalf("expected range format in output:\n%s", output)
	}
	if !strings.Contains(output, "90: async def build_map():") {
		t.Fatalf("expected single-line format in output:\n%s", output)
	}
}

func TestGenerate_TreeFormat(t *testing.T) {
	requireRipgrep(t)

	root := t.TempDir()
	writeProjectMapTestFile(t, root, "internal/agent/agent.go", "package agent\n\nfunc Run() {}\n")

	pm := buildProjectMapForTest(t, root, 4000)
	output := pm.Generate()
	for _, want := range []string{"## Project Map", "📂 internal/agent/", "└── 📄 agent.go", "3: func Run()"} {
		if !strings.Contains(output, want) {
			t.Fatalf("tree output missing %q:\n%s", want, output)
		}
	}
}

func TestGenerate_TestFilesIncluded(t *testing.T) {
	requireRipgrep(t)

	root := t.TempDir()
	writeProjectMapTestFile(t, root, "pkg/service_test.go", "package pkg\n\nfunc TestServiceBuild() {}\n")

	pm := buildProjectMapForTest(t, root, 4000)
	output := pm.Generate()
	if !strings.Contains(output, "service_test.go") {
		t.Fatalf("expected test file in output:\n%s", output)
	}
	if !strings.Contains(output, "func TestServiceBuild()") {
		t.Fatalf("expected test symbol in output:\n%s", output)
	}
}
