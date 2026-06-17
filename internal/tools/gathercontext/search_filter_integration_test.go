package gathercontext

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/locator"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

func TestGatherContext_SearchRouteKeepsTypeScriptFilterOnFallback(t *testing.T) {
	if !common.IsRipgrepAvailable() {
		t.Skip("ripgrep not available")
	}
	common.ResetRipgrepAvailabilityForTest()
	t.Cleanup(common.ResetRipgrepAvailabilityForTest)

	root := t.TempDir()
	withGatherContextWorkingDir(t, root)

	writeGatherContextFiles(t, map[string]string{
		filepath.Join(root, "src", "Button.tsx"): "export function Button() { return <button /> }\n",
	})

	result, _, err := (&Tool{}).Run(tools.ExecutionContext{
		Stdout:             io.Discard,
		Stderr:             io.Discard,
		LocatorRegistry:    locator.NewRegistry(),
		ProjectMapRootPath: root,
		InvocationCWD:      root,
	}, map[string]string{
		"query":       "Button",
		"path":        root,
		"file_filter": "typescript",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(result, "Route: Structured impact") || strings.Contains(result, "Recommended reads:") {
		t.Fatalf("file_filter=typescript should stay on fallback route, got:\n%s", result)
	}
	if !strings.Contains(result, "Route: Impact search") || !strings.Contains(result, "src/Button.tsx") {
		t.Fatalf("expected fallback impact search to retain TSX file, got:\n%s", result)
	}
}

func TestGatherContext_SearchRouteHonorsCjsFileFilterOnRipgrepPath(t *testing.T) {
	if !common.IsRipgrepAvailable() {
		t.Skip("ripgrep not available")
	}
	common.ResetRipgrepAvailabilityForTest()
	t.Cleanup(common.ResetRipgrepAvailabilityForTest)

	root := t.TempDir()
	withGatherContextWorkingDir(t, root)

	pkgDir := filepath.Join(root, "pkg")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "main.cjs"), []byte("// target cjs marker\nmodule.exports = {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "main.js"), []byte("// target cjs marker\nexport default {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, _, err := (&Tool{}).Run(tools.ExecutionContext{
		Stdout:             io.Discard,
		Stderr:             io.Discard,
		LocatorRegistry:    locator.NewRegistry(),
		ProjectMapRootPath: root,
		InvocationCWD:      root,
	}, map[string]string{
		"query":       "target cjs marker",
		"path":        "pkg",
		"file_filter": "cjs",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(result, "unrecognized file type") {
		t.Fatalf("expected cjs file_filter to avoid rg type error, got:\n%s", result)
	}
	if strings.Contains(result, "Route: Direct read") || strings.Contains(result, "Route: Directory listing") {
		t.Fatalf("expected cjs file_filter query to stay off direct routes, got:\n%s", result)
	}
	if !strings.Contains(result, "pkg/main.cjs") {
		t.Fatalf("expected cjs file_filter query to include main.cjs, got:\n%s", result)
	}
	if strings.Contains(result, "pkg/main.js") {
		t.Fatalf("expected cjs file_filter query to exclude main.js, got:\n%s", result)
	}
}

func TestGatherContext_SearchRouteHonorsPythonStubFileFilterOnRipgrepPath(t *testing.T) {
	if !common.IsRipgrepAvailable() {
		t.Skip("ripgrep not available")
	}
	common.ResetRipgrepAvailabilityForTest()
	t.Cleanup(common.ResetRipgrepAvailabilityForTest)

	root := t.TempDir()
	withGatherContextWorkingDir(t, root)

	pkgDir := filepath.Join(root, "pkg")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "types.pyi"), []byte("PYI_MARKER = 'stub-surface'\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "runtime.py"), []byte("PY_MARKER = 'runtime-surface'\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "main.go"), []byte("const marker = \"stub-surface\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, _, err := (&Tool{}).Run(tools.ExecutionContext{
		Stdout:             io.Discard,
		Stderr:             io.Discard,
		LocatorRegistry:    locator.NewRegistry(),
		ProjectMapRootPath: root,
		InvocationCWD:      root,
	}, map[string]string{
		"query":       "stub-surface",
		"path":        "pkg",
		"file_filter": "py",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(result, "Route: Direct read") || strings.Contains(result, "Route: Directory listing") {
		t.Fatalf("expected python file_filter text query to stay on search route, got:\n%s", result)
	}
	if !strings.Contains(result, "pkg/types.pyi") {
		t.Fatalf("expected python file_filter query to include types.pyi, got:\n%s", result)
	}
	if strings.Contains(result, "pkg/main.go") {
		t.Fatalf("expected python file_filter query to exclude go files, got:\n%s", result)
	}
}

func TestGatherContext_SearchRouteHonorsBroadenedLanguageFileFilterContract(t *testing.T) {
	if !common.IsRipgrepAvailable() {
		t.Skip("ripgrep not available")
	}
	common.ResetRipgrepAvailabilityForTest()
	t.Cleanup(common.ResetRipgrepAvailabilityForTest)

	root := t.TempDir()
	withGatherContextWorkingDir(t, root)

	pkgDir := filepath.Join(root, "pkg")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	files := map[string]string{
		filepath.Join(pkgDir, "native.h"):               "#define TARGET_MACRO 1\n",
		filepath.Join(pkgDir, "application.properties"): "app.name=TARGET_PROPERTY\n",
		filepath.Join(pkgDir, "shell.zsh"):              "echo TARGET_SHELL\n",
		filepath.Join(pkgDir, "main.go"):                "const marker = \"noise\"\n",
	}
	for path, body := range files {
		if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	execCtx := tools.ExecutionContext{
		Stdout:             io.Discard,
		Stderr:             io.Discard,
		LocatorRegistry:    locator.NewRegistry(),
		ProjectMapRootPath: root,
		InvocationCWD:      root,
	}

	tests := []struct {
		name        string
		query       string
		fileFilter  string
		wantPath    string
		wantExclude string
	}{
		{name: "c includes headers", query: "TARGET_MACRO", fileFilter: "c", wantPath: "pkg/native.h", wantExclude: "pkg/main.go"},
		{name: "java includes properties", query: "TARGET_PROPERTY", fileFilter: "java", wantPath: "pkg/application.properties", wantExclude: "pkg/main.go"},
		{name: "sh includes zsh", query: "TARGET_SHELL", fileFilter: "sh", wantPath: "pkg/shell.zsh", wantExclude: "pkg/main.go"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, _, err := (&Tool{}).Run(execCtx, map[string]string{
				"query":       tt.query,
				"path":        "pkg",
				"file_filter": tt.fileFilter,
			})
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if strings.Contains(result, "Route: Direct read") || strings.Contains(result, "Route: Directory listing") {
				t.Fatalf("expected %s query to stay on search route, got:\n%s", tt.fileFilter, result)
			}
			if !strings.Contains(result, tt.wantPath) {
				t.Fatalf("expected %s file_filter query to include %s, got:\n%s", tt.fileFilter, tt.wantPath, result)
			}
			if strings.Contains(result, tt.wantExclude) {
				t.Fatalf("expected %s file_filter query to exclude unrelated files, got:\n%s", tt.fileFilter, result)
			}
		})
	}
}

func TestGatherContext_AbsoluteScopedPathPreservesWorkspaceRelativeFileFilter(t *testing.T) {
	if !common.IsRipgrepAvailable() {
		t.Skip("ripgrep not available")
	}
	common.ResetRipgrepAvailabilityForTest()
	t.Cleanup(common.ResetRipgrepAvailabilityForTest)

	root := t.TempDir()
	withGatherContextWorkingDir(t, root)

	pkgDir := filepath.Join(root, "pkg")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "main.js"), []byte("const marker = 'absolute-basis-marker'\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pkgDir, "other.ts"), []byte("const marker = 'absolute-basis-marker'\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	execCtx := tools.ExecutionContext{
		Stdout:             io.Discard,
		Stderr:             io.Discard,
		LocatorRegistry:    locator.NewRegistry(),
		ProjectMapRootPath: root,
		InvocationCWD:      root,
	}

	run := func(path string) string {
		t.Helper()
		result, _, err := (&Tool{}).Run(execCtx, map[string]string{
			"query":       "absolute-basis-marker",
			"path":        path,
			"file_filter": "pkg/*.js",
		})
		if err != nil {
			t.Fatalf("unexpected error for path %q: %v", path, err)
		}
		return result
	}

	relativeResult := run("pkg")
	absoluteResult := run(pkgDir)

	for _, result := range []string{relativeResult, absoluteResult} {
		if strings.Contains(result, "No matches found") {
			t.Fatalf("expected workspace-relative glob search to match under both path forms, got:\n%s", result)
		}
		if !strings.Contains(result, "pkg/main.js") {
			t.Fatalf("expected workspace-relative glob search to include pkg/main.js, got:\n%s", result)
		}
		if strings.Contains(result, "pkg/other.ts") {
			t.Fatalf("expected workspace-relative glob search to exclude pkg/other.ts, got:\n%s", result)
		}
	}
}
