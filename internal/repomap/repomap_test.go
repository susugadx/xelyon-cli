package repomap

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

func requireRipgrep(t *testing.T) {
	t.Helper()
	common.ResetRipgrepAvailabilityForTest()
	t.Cleanup(common.ResetRipgrepAvailabilityForTest)
	if !common.IsRipgrepAvailable() {
		t.Skip("ripgrep (rg) not available")
	}
}

func writeProjectMapTestFile(t *testing.T, root, relPath, content string) {
	t.Helper()
	path := filepath.Join(root, relPath)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("MkdirAll(%s) error = %v", relPath, err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("WriteFile(%s) error = %v", relPath, err)
	}
}

func buildProjectMapForTest(t *testing.T, root string, maxTokens int, ignoreDirs ...string) *ProjectMap {
	t.Helper()
	setProjectMapTestHome(t)
	pm := NewProjectMap(root, maxTokens, ignoreDirs...)
	if err := pm.Build(); err != nil {
		t.Fatalf("Build() error = %v", err)
	}
	return pm
}

func findFileEntry(t *testing.T, pm *ProjectMap, relPath string) *FileEntry {
	t.Helper()
	for _, file := range pm.Files {
		if file.Path == filepath.ToSlash(relPath) {
			return file
		}
	}
	t.Fatalf("file entry %s not found", relPath)
	return nil
}

func signatures(file *FileEntry) []string {
	result := make([]string, 0, len(file.Symbols))
	for _, symbol := range file.Symbols {
		result = append(result, symbol.Signature)
	}
	return result
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

func TestBuild_TypeScriptProject(t *testing.T) {
	requireRipgrep(t)

	root := t.TempDir()
	writeProjectMapTestFile(t, root, "src/app.ts", "export interface Config {}\nexport class Builder {}\nexport function buildMap() {}\n")

	pm := buildProjectMapForTest(t, root, 4000)
	file := findFileEntry(t, pm, "src/app.ts")
	got := strings.Join(signatures(file), "\n")
	for _, want := range []string{"export interface Config", "export class Builder", "export function buildMap()"} {
		if !strings.Contains(got, want) {
			t.Fatalf("TypeScript signatures missing %q from:\n%s", want, got)
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

func TestGenerate_WithGitStatus(t *testing.T) {
	requireRipgrep(t)

	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	root := t.TempDir()
	runGit := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
		}
	}

	runGit("init")
	writeProjectMapTestFile(t, root, "main.go", "package main\n\nfunc Build() {}\n")

	pm := buildProjectMapForTest(t, root, 4000)
	output := pm.Generate()
	if !strings.Contains(output, "## Uncommitted Changes") {
		t.Fatalf("expected git status section:\n%s", output)
	}
	if !strings.Contains(output, "?? main.go") {
		t.Fatalf("expected untracked file in git status:\n%s", output)
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
