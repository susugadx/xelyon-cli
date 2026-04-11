package gathercontext

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/locator"
	"github.com/susugadx/xelyon-cli/internal/repomap"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

func TestGatherContext_SearchRouteUsesStructuredImpactAndPrefetch(t *testing.T) {
	if !common.IsRipgrepAvailable() {
		t.Skip("ripgrep not available")
	}
	common.ResetRipgrepAvailabilityForTest()
	t.Cleanup(common.ResetRipgrepAvailabilityForTest)

	root := t.TempDir()
	withGatherContextWorkingDir(t, root)

	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "builder.go"), []byte(`package example

type Builder interface {
	Build() string
}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	projectMap := &repomap.ProjectMap{
		RootPath: root,
		Files: []*repomap.FileEntry{
			{
				Path: "builder.go",
				Symbols: []repomap.Symbol{
					{
						Name:      "Builder",
						Kind:      "interface",
						Line:      3,
						EndLine:   5,
						Signature: "type Builder interface { Build() string }",
						Exported:  true,
					},
				},
			},
		},
	}

	result, _, err := (&Tool{}).Run(tools.ExecutionContext{
		Stdout:             io.Discard,
		Stderr:             io.Discard,
		LocatorRegistry:    locator.NewRegistry(),
		ProjectMap:         projectMap,
		ProjectMapRootPath: root,
		ProjectMapStateKey: "gather-context-prefetch",
		InvocationCWD:      root,
	}, map[string]string{
		"query":       "Builder",
		"path":        root,
		"file_filter": "go",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "Route: Structured impact + prefetched evidence") {
		t.Fatalf("expected structured impact prefetch route, got:\n%s", result)
	}
	if !strings.Contains(result, "Search / Discovery") || !strings.Contains(result, "Prefetched Evidence") {
		t.Fatalf("expected merged investigation sections, got:\n%s", result)
	}
	if !strings.Contains(result, "Recommended reads:") {
		t.Fatalf("expected structured impact output, got:\n%s", result)
	}
	if !strings.Contains(result, "type Builder interface") {
		t.Fatalf("expected prefetched definition evidence, got:\n%s", result)
	}
}

func TestGatherContext_SearchRouteSkipsPrefetchForAmbiguousSymbol(t *testing.T) {
	if !common.IsRipgrepAvailable() {
		t.Skip("ripgrep not available")
	}
	common.ResetRipgrepAvailabilityForTest()
	t.Cleanup(common.ResetRipgrepAvailabilityForTest)

	root := t.TempDir()
	withGatherContextWorkingDir(t, root)

	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "build.go"), []byte("package example\n\nfunc Build() string { return \"\" }\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "config.go"), []byte("package example\n\ntype Config struct{}\n\nfunc (Config) Build() string { return \"\" }\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, _, err := (&Tool{}).Run(tools.ExecutionContext{
		Stdout:          io.Discard,
		Stderr:          io.Discard,
		LocatorRegistry: locator.NewRegistry(),
		InvocationCWD:   root,
	}, map[string]string{
		"query":       "Build",
		"path":        root,
		"file_filter": "go",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, `Multiple symbols matched "Build":`) {
		t.Fatalf("expected ambiguous candidate list, got:\n%s", result)
	}
	if strings.Contains(result, "Prefetched Evidence") {
		t.Fatalf("expected ambiguous search to avoid speculative prefetch, got:\n%s", result)
	}
}

func TestGatherContext_BarePathLikeCollisionFallsBackToSearch(t *testing.T) {
	if !common.IsRipgrepAvailable() {
		t.Skip("ripgrep not available")
	}
	common.ResetRipgrepAvailabilityForTest()
	t.Cleanup(common.ResetRipgrepAvailabilityForTest)

	root := t.TempDir()
	withGatherContextWorkingDir(t, root)

	if err := os.MkdirAll(filepath.Join(root, "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "pkg", "config.go"), []byte(`package pkg

type config struct{}
`), 0o644); err != nil {
		t.Fatal(err)
	}

	projectMap := &repomap.ProjectMap{
		RootPath: root,
		Files: []*repomap.FileEntry{
			{
				Path: "pkg/config.go",
				Symbols: []repomap.Symbol{
					{
						Name:    "config",
						Kind:    "type",
						Line:    3,
						EndLine: 3,
					},
				},
			},
		},
	}

	result, _, err := (&Tool{}).Run(tools.ExecutionContext{
		Stdout:             io.Discard,
		Stderr:             io.Discard,
		LocatorRegistry:    locator.NewRegistry(),
		ProjectMap:         projectMap,
		ProjectMapRootPath: root,
		ProjectMapStateKey: "gather-context-config-search",
		InvocationCWD:      root,
	}, map[string]string{
		"query":       "config",
		"path":        "pkg",
		"file_filter": "go",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(result, "Route: Directory listing") {
		t.Fatalf("expected bare collision query to stay on search route, got:\n%s", result)
	}
	if !strings.Contains(result, "Search / Discovery") || !strings.Contains(result, `Recommended reads:`) {
		t.Fatalf("expected search output for colliding bare query, got:\n%s", result)
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

func TestGatherContext_PackageLikeSlashQueriesStayOnSearchRoute(t *testing.T) {
	if !common.IsRipgrepAvailable() {
		t.Skip("ripgrep not available")
	}
	common.ResetRipgrepAvailabilityForTest()
	t.Cleanup(common.ResetRipgrepAvailabilityForTest)

	root := t.TempDir()
	withGatherContextWorkingDir(t, root)

	for _, dir := range []string{filepath.Join(root, "pkg"), filepath.Join(root, "pkg", "errors"), filepath.Join(root, "internal", "agent")} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "internal", "agent", "agent.go"), []byte("package agent\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "pkg", "imports.go"), []byte(`package pkg

import (
	"internal/agent"
	"pkg/errors"
	"github.com/foo/bar"
)
`), 0o644); err != nil {
		t.Fatal(err)
	}

	execCtx := tools.ExecutionContext{
		Stdout:             io.Discard,
		Stderr:             io.Discard,
		LocatorRegistry:    locator.NewRegistry(),
		ProjectMapRootPath: root,
		InvocationCWD:      root,
	}

	for _, query := range []string{filepath.Join("internal", "agent"), filepath.Join("pkg", "errors"), "github.com/foo/bar"} {
		result, _, err := (&Tool{}).Run(execCtx, map[string]string{
			"query":       query,
			"path":        ".",
			"file_filter": "go",
		})
		if err != nil {
			t.Fatalf("%q unexpected error: %v", query, err)
		}
		if strings.Contains(result, "Route: Direct query") || strings.Contains(result, "Error: direct path not found") {
			t.Fatalf("%q should fall back to search instead of direct error, got:\n%s", query, result)
		}
		if strings.Contains(result, "Route: Directory listing") || strings.Contains(result, "Route: Direct read") {
			t.Fatalf("%q should stay off direct routes even when matching directory exists, got:\n%s", query, result)
		}
		if !strings.Contains(result, "pkg/imports.go") || !strings.Contains(result, query) {
			t.Fatalf("%q expected search result hit, got:\n%s", query, result)
		}
	}
}

func TestGatherContext_UnresolvedBareLocatorLikeQueryFallsBackToSearch(t *testing.T) {
	if !common.IsRipgrepAvailable() {
		t.Skip("ripgrep not available")
	}
	common.ResetRipgrepAvailabilityForTest()
	t.Cleanup(common.ResetRipgrepAvailabilityForTest)

	root := t.TempDir()
	withGatherContextWorkingDir(t, root)

	if err := os.WriteFile(filepath.Join(root, "constants.go"), []byte(`package main

const L1 = "literal-locator-like"
`), 0o644); err != nil {
		t.Fatal(err)
	}

	result, _, err := (&Tool{}).Run(tools.ExecutionContext{
		Stdout:             io.Discard,
		Stderr:             io.Discard,
		LocatorRegistry:    locator.NewRegistry(),
		ProjectMapRootPath: root,
		InvocationCWD:      root,
	}, map[string]string{
		"query":       "L1",
		"path":        ".",
		"file_filter": "go",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(result, "Error: no valid locator IDs found") || strings.Contains(result, "Route: Direct read") {
		t.Fatalf("expected unresolved bare locator-like query to avoid locator direct read, got:\n%s", result)
	}
	if !strings.Contains(result, "constants.go") || !strings.Contains(result, "const L1") {
		t.Fatalf("expected unresolved bare locator-like query to fall back to search, got:\n%s", result)
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

func TestGatherContext_ScopedExactFilenameQueryUsesDirectRead(t *testing.T) {
	root := t.TempDir()
	withGatherContextWorkingDir(t, root)

	subdir := filepath.Join(root, "pkg")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "target.go"), []byte(`package main

const selected = "root"
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subdir, "target.go"), []byte(`package pkg

const selected = "subdir"
`), 0o644); err != nil {
		t.Fatal(err)
	}

	result, _, err := (&Tool{}).Run(tools.ExecutionContext{
		Stdout:             io.Discard,
		Stderr:             io.Discard,
		LocatorRegistry:    locator.NewRegistry(),
		ProjectMapRootPath: root,
		InvocationCWD:      root,
	}, map[string]string{
		"query":       "target.go",
		"path":        "pkg",
		"file_filter": "go",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "Route: Direct read") {
		t.Fatalf("expected scoped exact filename query to use direct read, got:\n%s", result)
	}
	if strings.Contains(result, "Route: Direct query") {
		t.Fatalf("expected scoped exact filename query to avoid direct error routing, got:\n%s", result)
	}
	if strings.Contains(result, "No matches found") {
		t.Fatalf("expected scoped exact filename query to avoid search fallback miss, got:\n%s", result)
	}
	if !strings.Contains(result, "📄 File: pkg/target.go") || !strings.Contains(result, `"subdir"`) {
		t.Fatalf("expected direct read result from pkg target, got:\n%s", result)
	}
	if strings.Contains(result, `"root"`) {
		t.Fatalf("expected scoped direct read to avoid root shadow file, got:\n%s", result)
	}
}

func TestGatherContext_ScopedBareFilenameMissReturnsDirectError(t *testing.T) {
	root := t.TempDir()
	withGatherContextWorkingDir(t, root)

	subdir := filepath.Join(root, "pkg")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "sample.go"), []byte("package main\nconst selected = \"root-shadow\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "notes.go"), []byte("package main\n// sample.go root-hit\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subdir, "notes.go"), []byte("package pkg\n// sample.go scoped-hit\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, _, err := (&Tool{}).Run(tools.ExecutionContext{
		Stdout:             io.Discard,
		Stderr:             io.Discard,
		LocatorRegistry:    locator.NewRegistry(),
		ProjectMapRootPath: root,
		InvocationCWD:      root,
	}, map[string]string{
		"query":       "sample.go",
		"path":        "pkg",
		"file_filter": "go",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "Route: Direct query") {
		t.Fatalf("expected scoped bare filename miss to stay on direct route, got:\n%s", result)
	}
	if !strings.Contains(result, "Error: direct path not found: sample.go") {
		t.Fatalf("expected scoped bare filename miss to surface direct error, got:\n%s", result)
	}
	if strings.Contains(result, "root-hit") || strings.Contains(result, "root-shadow") || strings.Contains(result, "scoped-hit") {
		t.Fatalf("expected scoped bare filename miss to avoid unrelated fallback results, got:\n%s", result)
	}
	if strings.Contains(result, "No matches found") || strings.Contains(result, "Route: Auto search") {
		t.Fatalf("expected scoped bare filename miss to avoid search fallback, got:\n%s", result)
	}
}
