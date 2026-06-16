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

func TestGatherContext_SearchRouteUsesGoStructuredImpactForScopedGlob(t *testing.T) {
	if !common.IsRipgrepAvailable() {
		t.Skip("ripgrep not available")
	}
	common.ResetRipgrepAvailabilityForTest()
	t.Cleanup(common.ResetRipgrepAvailabilityForTest)

	root := t.TempDir()
	withGatherContextWorkingDir(t, root)

	writeGatherContextFiles(t, map[string]string{
		filepath.Join(root, "packages", "app", "src", "build.go"):   "package app\n\nfunc ScopedImpactBuild() string { return \"app\" }\n",
		filepath.Join(root, "packages", "other", "src", "build.go"): "package other\n\nfunc ScopedImpactBuild() string { return \"other\" }\n",
	})

	result, _ := runGatherContext(t, newGatherContextExecCtx(root), map[string]string{
		"query":       "ScopedImpactBuild",
		"path":        root,
		"file_filter": "packages/app/src/**/*.go",
	})

	assertGatherContextContainsAll(t, result,
		"Route: Structured impact + prefetched evidence",
		"Recommended reads:",
		"Prefetched Evidence",
		"packages/app/src/build.go",
		`return "app"`,
	)
	assertGatherContextExcludesAll(t, result,
		"packages/other/src/build.go",
		`return "other"`,
	)
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
	if !strings.Contains(result, "Prefetch skipped: ambiguous structured impact") {
		t.Fatalf("expected ambiguous prefetch skip note, got:\n%s", result)
	}
}

func TestGatherContext_SearchRouteUsesTypeScriptStructuredImpactAndPrefetch(t *testing.T) {
	if !common.IsRipgrepAvailable() {
		t.Skip("ripgrep not available")
	}
	common.ResetRipgrepAvailabilityForTest()
	t.Cleanup(common.ResetRipgrepAvailabilityForTest)

	root := t.TempDir()
	withGatherContextWorkingDir(t, root)

	writeGatherContextFiles(t, map[string]string{
		filepath.Join(root, "src", "build.ts"):      "export function buildUser(id: string) { return id }\n",
		filepath.Join(root, "src", "app.ts"):        "import { buildUser } from './build'\nbuildUser('1')\n",
		filepath.Join(root, "src", "build.test.ts"): "import { buildUser } from './build'\nbuildUser('test')\n",
	})

	result, _, err := (&Tool{}).Run(tools.ExecutionContext{
		Stdout:             io.Discard,
		Stderr:             io.Discard,
		LocatorRegistry:    locator.NewRegistry(),
		ProjectMapRootPath: root,
		InvocationCWD:      root,
	}, map[string]string{
		"query":       "buildUser",
		"path":        root,
		"file_filter": "ts",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "Route: Structured impact + prefetched evidence") {
		t.Fatalf("expected TypeScript structured impact prefetch route, got:\n%s", result)
	}
	if !strings.Contains(result, "Search / Discovery") || !strings.Contains(result, "Prefetched Evidence") {
		t.Fatalf("expected merged TypeScript investigation sections, got:\n%s", result)
	}
	if !strings.Contains(result, "Recommended reads:") {
		t.Fatalf("expected TypeScript structured impact output, got:\n%s", result)
	}
	if !strings.Contains(result, "Diagnostics: resolved_by=ast, confidence=medium") {
		t.Fatalf("expected TypeScript diagnostics summary in search discovery, got:\n%s", result)
	}
	if !strings.Contains(result, "export function buildUser") {
		t.Fatalf("expected prefetched TypeScript definition evidence, got:\n%s", result)
	}
	if !strings.Contains(result, "Prefetch limited: confidence=medium") {
		t.Fatalf("expected medium-confidence TypeScript prefetch limit note, got:\n%s", result)
	}
	assertGatherContextPrefetchedFileCount(t, result, 2)
}

func TestGatherContext_SearchRouteSkipsPrefetchForAmbiguousTypeScriptSymbol(t *testing.T) {
	if !common.IsRipgrepAvailable() {
		t.Skip("ripgrep not available")
	}
	common.ResetRipgrepAvailabilityForTest()
	t.Cleanup(common.ResetRipgrepAvailabilityForTest)

	root := t.TempDir()
	withGatherContextWorkingDir(t, root)

	writeGatherContextFiles(t, map[string]string{
		filepath.Join(root, "src", "a.ts"): "export function buildUser(id: string) { return id }\n",
		filepath.Join(root, "src", "b.ts"): "export function buildUser(id: string) { return id }\n",
	})

	result, _, err := (&Tool{}).Run(tools.ExecutionContext{
		Stdout:             io.Discard,
		Stderr:             io.Discard,
		LocatorRegistry:    locator.NewRegistry(),
		ProjectMapRootPath: root,
		InvocationCWD:      root,
	}, map[string]string{
		"query":       "buildUser",
		"path":        root,
		"file_filter": "ts",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, `Multiple definitions found for "buildUser":`) {
		t.Fatalf("expected ambiguous TypeScript candidate list, got:\n%s", result)
	}
	if strings.Contains(result, "Prefetched Evidence") {
		t.Fatalf("expected ambiguous TypeScript search to avoid speculative prefetch, got:\n%s", result)
	}
	if !strings.Contains(result, "Prefetch skipped: ambiguous structured impact") {
		t.Fatalf("expected ambiguous TypeScript prefetch skip note, got:\n%s", result)
	}
}

func TestGatherContext_SearchRouteUsesTSXStructuredImpactAndPrefetch(t *testing.T) {
	if !common.IsRipgrepAvailable() {
		t.Skip("ripgrep not available")
	}
	common.ResetRipgrepAvailabilityForTest()
	t.Cleanup(common.ResetRipgrepAvailabilityForTest)

	root := t.TempDir()
	withGatherContextWorkingDir(t, root)

	writeGatherContextFiles(t, map[string]string{
		filepath.Join(root, "src", "Button.tsx"):      "export function Button() { return <button /> }\n",
		filepath.Join(root, "src", "App.tsx"):         "import { Button } from './Button'\nexport function App() { return <Button /> }\n",
		filepath.Join(root, "src", "Button.test.tsx"): "import { Button } from './Button'\nit('renders', () => <Button />)\n",
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
		"file_filter": "tsx",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "Route: Structured impact + prefetched evidence") {
		t.Fatalf("expected TSX structured impact prefetch route, got:\n%s", result)
	}
	if !strings.Contains(result, "Recommended reads:") || !strings.Contains(result, "Prefetched Evidence") {
		t.Fatalf("expected TSX structured impact with prefetched evidence, got:\n%s", result)
	}
	if !strings.Contains(result, "export function Button") || !strings.Contains(result, "<Button />") {
		t.Fatalf("expected prefetched TSX evidence, got:\n%s", result)
	}
}

func TestGatherContext_SearchRouteSkipsPrefetchForAmbiguousTSXSymbol(t *testing.T) {
	if !common.IsRipgrepAvailable() {
		t.Skip("ripgrep not available")
	}
	common.ResetRipgrepAvailabilityForTest()
	t.Cleanup(common.ResetRipgrepAvailabilityForTest)

	root := t.TempDir()
	withGatherContextWorkingDir(t, root)

	writeGatherContextFiles(t, map[string]string{
		filepath.Join(root, "src", "Button.tsx"): "export function Button() { return <button /> }\n",
		filepath.Join(root, "src", "Panel.tsx"):  "export function Button() { return <section /> }\n",
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
		"file_filter": "tsx",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, `Multiple definitions found for "Button":`) {
		t.Fatalf("expected ambiguous TSX candidate list, got:\n%s", result)
	}
	if strings.Contains(result, "Prefetched Evidence") {
		t.Fatalf("expected ambiguous TSX search to avoid speculative prefetch, got:\n%s", result)
	}
}

func TestGatherContext_SearchRouteUsesJavaScriptStructuredImpactAndPrefetch(t *testing.T) {
	if !common.IsRipgrepAvailable() {
		t.Skip("ripgrep not available")
	}
	common.ResetRipgrepAvailabilityForTest()
	t.Cleanup(common.ResetRipgrepAvailabilityForTest)

	root := t.TempDir()
	withGatherContextWorkingDir(t, root)

	writeGatherContextFiles(t, map[string]string{
		filepath.Join(root, "src", "build.js"):      "export function buildUser(id) { return id }\n",
		filepath.Join(root, "src", "app.js"):        "import { buildUser } from './build.js'\nbuildUser('1')\n",
		filepath.Join(root, "src", "build.test.js"): "import { buildUser } from './build.js'\nbuildUser('test')\n",
	})

	result, _, err := (&Tool{}).Run(tools.ExecutionContext{
		Stdout:             io.Discard,
		Stderr:             io.Discard,
		LocatorRegistry:    locator.NewRegistry(),
		ProjectMapRootPath: root,
		InvocationCWD:      root,
	}, map[string]string{
		"query":       "buildUser",
		"path":        root,
		"file_filter": "js",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "Route: Structured impact + prefetched evidence") {
		t.Fatalf("expected JavaScript structured impact prefetch route, got:\n%s", result)
	}
	if !strings.Contains(result, "Recommended reads:") || !strings.Contains(result, "Prefetched Evidence") {
		t.Fatalf("expected JavaScript structured impact with prefetched evidence, got:\n%s", result)
	}
	if !strings.Contains(result, "Diagnostics: resolved_by=ast, confidence=medium") {
		t.Fatalf("expected JavaScript diagnostics summary in search discovery, got:\n%s", result)
	}
	if !strings.Contains(result, "export function buildUser") || !strings.Contains(result, "buildUser('1')") {
		t.Fatalf("expected prefetched JavaScript evidence, got:\n%s", result)
	}
	if !strings.Contains(result, "Prefetch limited: confidence=medium") {
		t.Fatalf("expected medium-confidence JavaScript prefetch limit note, got:\n%s", result)
	}
	assertGatherContextPrefetchedFileCount(t, result, 2)
}

func TestGatherContext_SearchRouteSkipsPrefetchForAmbiguousJavaScriptSymbol(t *testing.T) {
	if !common.IsRipgrepAvailable() {
		t.Skip("ripgrep not available")
	}
	common.ResetRipgrepAvailabilityForTest()
	t.Cleanup(common.ResetRipgrepAvailabilityForTest)

	root := t.TempDir()
	withGatherContextWorkingDir(t, root)

	writeGatherContextFiles(t, map[string]string{
		filepath.Join(root, "src", "a.js"): "export function buildUser(id) { return id }\n",
		filepath.Join(root, "src", "b.js"): "export function buildUser(id) { return id }\n",
	})

	result, _, err := (&Tool{}).Run(tools.ExecutionContext{
		Stdout:             io.Discard,
		Stderr:             io.Discard,
		LocatorRegistry:    locator.NewRegistry(),
		ProjectMapRootPath: root,
		InvocationCWD:      root,
	}, map[string]string{
		"query":       "buildUser",
		"path":        root,
		"file_filter": "js",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, `Multiple definitions found for "buildUser":`) {
		t.Fatalf("expected ambiguous JavaScript candidate list, got:\n%s", result)
	}
	if strings.Contains(result, "Prefetched Evidence") {
		t.Fatalf("expected ambiguous JavaScript search to avoid speculative prefetch, got:\n%s", result)
	}
	if !strings.Contains(result, "Prefetch skipped: ambiguous structured impact") {
		t.Fatalf("expected ambiguous JavaScript prefetch skip note, got:\n%s", result)
	}
}
