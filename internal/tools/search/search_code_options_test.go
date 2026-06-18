package search

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/config"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

func TestSearchCode_FileTypePreferred(t *testing.T) {
	setupSearchTestMocks(t)

	dir := t.TempDir()
	goFile := filepath.Join(dir, "typed.go")
	jsFile := filepath.Join(dir, "typed.js")
	if err := os.WriteFile(goFile, []byte("func typedTarget() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(jsFile, []byte("function typedTarget() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result := ExecuteSearchCode(SearchOptions{Pattern: "typedTarget", Path: dir, FilePattern: "*.js", FileType: "go", CtxLines: 0, TokenBudget: 3000, IsRegex: true, Multiline: false})
	if !strings.Contains(result, "typed.go") {
		t.Fatalf("expected go file in result, got: %s", result)
	}
	if strings.Contains(result, "typed.js") {
		t.Fatalf("file_type should take precedence over file_pattern, got: %s", result)
	}
}

func TestSearchCode_FixedStrings(t *testing.T) {
	setupSearchTestMocks(t)

	dir := t.TempDir()
	file1 := filepath.Join(dir, "fixed.go")
	if err := os.WriteFile(file1, []byte("var name = \"a+b\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result := ExecuteSearchCode(SearchOptions{Pattern: "a+b", Mode: string(SearchModeLiteral), Path: dir, FilePattern: "*.go", FileType: "", CtxLines: 0, TokenBudget: 3000, IsRegex: false, Multiline: false})
	if strings.Contains(result, "No matches found") || !strings.Contains(result, "a+b") {
		t.Fatalf("expected literal match with is_regex=false, got: %s", result)
	}
}

func TestSearchCode_IncludeHidden(t *testing.T) {
	setupSearchTestMocks(t)

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".env"), []byte("SECRET=hidden_value\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result := ExecuteSearchCode(SearchOptions{
		Pattern:     "hidden_value",
		Path:        dir,
		IsRegex:     true,
		CtxLines:    -1,
		TokenBudget: -1,
	})
	if strings.Contains(result, ".env") {
		t.Fatalf("hidden files should be excluded by default, got: %s", result)
	}

	result = ExecuteSearchCode(SearchOptions{
		Pattern:       "hidden_value",
		Path:          dir,
		IsRegex:       true,
		IncludeHidden: true,
		CtxLines:      -1,
		TokenBudget:   -1,
	})
	if !strings.Contains(result, ".env") {
		t.Fatalf("hidden files should be included with IncludeHidden, got: %s", result)
	}
}

func TestSearchCode_GrepFallback_DoesNotExcludeRootDot(t *testing.T) {
	setupSearchTestMocks(t)

	if runtime.GOOS == "windows" {
		t.Skip("grep fallback regression test is linux/mac specific")
	}

	grepPath, err := exec.LookPath("grep")
	if err != nil {
		t.Skip("grep not available")
	}

	binDir := t.TempDir()
	if err := os.Symlink(grepPath, filepath.Join(binDir, "grep")); err != nil {
		t.Skipf("failed to prepare isolated grep PATH: %v", err)
	}

	dir := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(oldwd)
	})

	t.Setenv("PATH", binDir)

	file1 := filepath.Join(dir, "search_target.go")
	if err := os.WriteFile(file1, []byte("package main\n\nfunc maybeAutoCompress() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result := ExecuteSearchCode(SearchOptions{
		Pattern:     "maybeAutoCompress",
		Path:        ".",
		FilePattern: "*.go",
		CtxLines:    0,
		TokenBudget: 3000,
		IsRegex:     true,
		Multiline:   false,
	})

	if strings.Contains(result, "No matches found") {
		t.Fatalf("expected grep fallback to find match from root dot, got: %s", result)
	}
	if strings.Contains(result, "Warning: ripgrep (rg) not found; using grep fallback mode.") {
		t.Fatalf("unexpected per-call grep fallback warning, got: %s", result)
	}
	if !strings.Contains(result, "search_target.go") {
		t.Fatalf("expected file name in result, got: %s", result)
	}
}

func TestResolveSearchPathBasisForOptions_UsesWorkspaceRootForAbsoluteScopedPath(t *testing.T) {
	root := t.TempDir()
	scopeDir := filepath.Join(root, "pkg")

	got := resolveSearchPathBasisForOptions(SearchOptions{
		Path:               scopeDir,
		ProjectMapRootPath: root,
		InvocationCWD:      root,
	})
	if got.Workdir != root || got.Target != "pkg" || got.MatchRoot != root {
		t.Fatalf("resolveSearchPathBasisForOptions(%q) = %+v, want workdir=%q target=%q matchRoot=%q", scopeDir, got, root, "pkg", root)
	}
}

func TestSearchCode_CjsFileFilterUsesRipgrepGlobs(t *testing.T) {
	setupSearchTestMocks(t)
	if !common.IsRipgrepAvailable() {
		t.Skip("ripgrep not available")
	}

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "main.cjs"), []byte("const marker = 'cjs-only-marker'\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "main.js"), []byte("const marker = 'cjs-only-marker'\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result := ExecuteSearchCode(SearchOptions{
		Pattern:     "cjs-only-marker",
		Mode:        string(SearchModeLiteral),
		Path:        dir,
		FileType:    "cjs",
		CtxLines:    0,
		TokenBudget: 3000,
	})
	if strings.Contains(result, "unrecognized file type") {
		t.Fatalf("expected cjs file_filter to avoid rg type error, got: %s", result)
	}
	if !strings.Contains(result, "main.cjs") {
		t.Fatalf("expected cjs file_filter to include main.cjs, got: %s", result)
	}
	if strings.Contains(result, "main.js") {
		t.Fatalf("expected cjs file_filter to exclude main.js, got: %s", result)
	}
}

func TestExecuteSearchCodeWithConfig_ProjectIgnorePatterns(t *testing.T) {
	setupSearchTestMocks(t)

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "xelyon.yaml"), []byte("ignore:\n  patterns:\n    - generated\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "generated"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "generated", "skip.go"), []byte("package generated\n\nfunc target() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "keep.go"), []byte("package main\n\nfunc target() {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldwd) })

	result := ExecuteSearchCodeWithConfig(config.DefaultConfig(), nil, SearchOptions{
		Pattern:     "target",
		Path:        dir,
		FilePattern: "*.go",
		CtxLines:    0,
		TokenBudget: 3000,
		IsRegex:     true,
	})

	if strings.Contains(result, "generated/skip.go") {
		t.Fatalf("generated/skip.go should be ignored by xelyon.yaml ignore.patterns, got %q", result)
	}
	if !strings.Contains(result, "keep.go") {
		t.Fatalf("keep.go should be included, got %q", result)
	}
}
