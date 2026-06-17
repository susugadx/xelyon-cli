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
