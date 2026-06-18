package directquery

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestResolveDirectQueryInput_ExactIgnoredTreePathBypassesIgnores(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "node_modules", "dep"), 0o755); err != nil {
		t.Fatal(err)
	}
	depPackage := filepath.Join(root, "node_modules", "dep", "package.json")
	if err := os.WriteFile(depPackage, []byte("{\"name\":\"dep\"}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	input, ok := parseDirectQueryInput(filepath.Join("node_modules", "dep", "package.json"))
	if !ok {
		t.Fatal("expected exact ignored-tree path query to parse")
	}

	resolution, errResult := resolveDirectQueryInput(tools.ExecutionContext{
		ProjectMapRootPath: root,
		InvocationCWD:      root,
	}, input)
	if errResult != "" {
		t.Fatalf("expected exact ignored-tree path to resolve, got %q", errResult)
	}
	if resolution.Kind != directQueryResolutionFiles {
		t.Fatalf("resolution.Kind = %q, want %q", resolution.Kind, directQueryResolutionFiles)
	}
	if len(resolution.Targets) != 1 {
		t.Fatalf("len(resolution.Targets) = %d, want 1", len(resolution.Targets))
	}
	target := resolution.Targets[0]
	if target.ResolvedPath != depPackage {
		t.Fatalf("target.ResolvedPath = %q, want %q", target.ResolvedPath, depPackage)
	}
	if target.RawEntry != filepath.Join("node_modules", "dep", "package.json") {
		t.Fatalf("target.RawEntry = %q, want dependency package.json entry", target.RawEntry)
	}
	if !target.BypassIgnores {
		t.Fatal("expected exact ignored-tree path target to bypass ignores")
	}
}

func TestResolveDirectQueryInput_ExactIgnoredTreeRangeBypassesIgnores(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "node_modules", "dep"), 0o755); err != nil {
		t.Fatal(err)
	}
	depPackage := filepath.Join(root, "node_modules", "dep", "package.json")
	if err := os.WriteFile(depPackage, []byte("{\n  \"name\": \"dep\"\n}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	input, ok := parseDirectQueryInput(filepath.Join("node_modules", "dep", "package.json:2-2"))
	if !ok {
		t.Fatal("expected exact ignored-tree range query to parse")
	}

	resolution, errResult := resolveDirectQueryInput(tools.ExecutionContext{
		ProjectMapRootPath: root,
		InvocationCWD:      root,
	}, input)
	if errResult != "" {
		t.Fatalf("expected exact ignored-tree range to resolve, got %q", errResult)
	}
	if resolution.Kind != directQueryResolutionFiles || len(resolution.Targets) != 1 {
		t.Fatalf("unexpected resolution: %+v", resolution)
	}
	target := resolution.Targets[0]
	if target.ResolvedPath != depPackage {
		t.Fatalf("target.ResolvedPath = %q, want %q", target.ResolvedPath, depPackage)
	}
	if !target.BypassIgnores {
		t.Fatal("expected exact ignored-tree range target to bypass ignores")
	}
	if target.StartLine != 2 || target.EndLine != 2 {
		t.Fatalf("target range = %d-%d, want 2-2", target.StartLine, target.EndLine)
	}
}
