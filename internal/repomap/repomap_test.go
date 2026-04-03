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

func buildProjectManifestForTest(t *testing.T, root string, maxTokens int, ignoreDirs ...string) *ProjectMap {
	t.Helper()
	setProjectMapTestHome(t)
	pm := NewProjectMap(root, maxTokens, ignoreDirs...)
	if err := pm.BuildManifest(); err != nil {
		t.Fatalf("BuildManifest() error = %v", err)
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

func findSymbol(t *testing.T, file *FileEntry, name string) Symbol {
	t.Helper()
	for _, symbol := range file.Symbols {
		if symbol.Name == name {
			return symbol
		}
	}
	t.Fatalf("symbol %s not found in %s", name, file.Path)
	return Symbol{}
}

func signatures(file *FileEntry) []string {
	result := make([]string, 0, len(file.Symbols))
	for _, symbol := range file.Symbols {
		result = append(result, symbol.Signature)
	}
	return result
}

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

func TestBuildManifest_GenerateManifest(t *testing.T) {
	requireRipgrep(t)

	root := t.TempDir()
	writeProjectMapTestFile(t, root, "README.md", "# test\n")
	writeProjectMapTestFile(t, root, "main.go", "package main\n")
	writeProjectMapTestFile(t, root, "internal/agent/compress.go", "package agent\n")
	writeProjectMapTestFile(t, root, "internal/config/project.go", "package config\n")

	pm := buildProjectManifestForTest(t, root, 4000)
	output := pm.GenerateManifest([]string{"internal/agent"})

	if !strings.Contains(output, "Top-level directories:") {
		t.Fatalf("manifest should contain top-level directories:\n%s", output)
	}
	if !strings.Contains(output, "- internal/") {
		t.Fatalf("manifest should contain internal/ directory:\n%s", output)
	}
	if !strings.Contains(output, "Top-level files:") || !strings.Contains(output, "- README.md") {
		t.Fatalf("manifest should contain top-level files:\n%s", output)
	}
	if !strings.Contains(output, "Priority files:") || !strings.Contains(output, "internal/agent/compress.go") {
		t.Fatalf("manifest should contain prioritized file:\n%s", output)
	}
	if strings.Contains(output, "func ") {
		t.Fatalf("manifest should stay lightweight without symbols:\n%s", output)
	}
}

func TestGenerateManifest_NilPrioritizedPathsReturnsLightweightManifest(t *testing.T) {
	requireRipgrep(t)

	root := t.TempDir()
	writeProjectMapTestFile(t, root, "README.md", "# test\n")
	writeProjectMapTestFile(t, root, "main.go", "package main\n\nfunc Build() {}\n")
	writeProjectMapTestFile(t, root, "internal/agent/compress.go", "package agent\n\nfunc Compress() {}\n")

	pm := buildProjectMapForTest(t, root, 4000)
	full := pm.Generate()
	manifest := pm.GenerateManifest(nil)

	if manifest == "" {
		t.Fatal("expected non-empty manifest")
	}
	if !strings.Contains(manifest, "Top-level directories:") {
		t.Fatalf("expected top-level directories in manifest:\n%s", manifest)
	}
	if !strings.Contains(manifest, "Top-level files:") {
		t.Fatalf("expected top-level files in manifest:\n%s", manifest)
	}
	if strings.Contains(manifest, "Priority files:") {
		t.Fatalf("stable base manifest should not include priority files:\n%s", manifest)
	}
	if strings.Contains(manifest, "func Build()") || strings.Contains(manifest, "func Compress()") {
		t.Fatalf("manifest should omit symbol dump:\n%s", manifest)
	}
	if len(manifest) >= len(full) {
		t.Fatalf("expected manifest to stay lighter than full map\nmanifest:\n%s\n\nfull:\n%s", manifest, full)
	}
}

func TestBuildManifest_RespectsFileGlobIgnorePatterns(t *testing.T) {
	requireRipgrep(t)

	root := t.TempDir()
	writeProjectMapTestFile(t, root, "assets/app.min.js", "console.log('skip')\n")
	writeProjectMapTestFile(t, root, "assets/app.js", "console.log('keep')\n")

	pm := buildProjectManifestForTest(t, root, 4000, "*.min.js")
	output := pm.GenerateManifest([]string{"assets"})

	if strings.Contains(output, "app.min.js") {
		t.Fatalf("file glob ignore should exclude app.min.js:\n%s", output)
	}
	if !strings.Contains(output, "app.js") {
		t.Fatalf("expected non-ignored file to remain:\n%s", output)
	}
}

func TestGenerateManifest_StaysWithinBudgetWithManyChanges(t *testing.T) {
	pm := &ProjectMap{
		MaxTokens: 40,
		Files: []*FileEntry{
			{Path: "README.md"},
			{Path: "Makefile"},
			{Path: "internal/agent/compress.go"},
			{Path: "internal/config/project.go"},
		},
	}
	for i := 0; i < 30; i++ {
		pm.GitStatus = append(pm.GitStatus, GitChange{
			Status: "M",
			Path:   filepath.ToSlash(filepath.Join("internal", "agent", "file"+strconv.Itoa(i)+".go")),
		})
	}

	output := pm.GenerateManifest([]string{"internal/agent"})
	if !pm.fitsBudget(output) {
		t.Fatalf("manifest must stay within budget, got:\n%s", output)
	}
	if strings.Count(output, "\n- M ") >= len(pm.GitStatus) {
		t.Fatalf("expected uncommitted changes to be trimmed under budget:\n%s", output)
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
