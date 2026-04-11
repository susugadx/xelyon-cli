package file

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestPlanGatherContextDirectRoute_Policy(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "sample.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.ex"), []byte("defmodule Main do\nend\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "main.exs"), []byte("IO.puts(\"hello\")\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "types.pyi"), []byte("class User: ...\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Makefile"), []byte("all:\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	execCtx := tools.ExecutionContext{
		ProjectMapRootPath: root,
		InvocationCWD:      root,
	}

	explicit := PlanGatherContextDirectRoute(execCtx, "config/", GatherContextDirectRoutePolicy{})
	if explicit.Kind != GatherContextDirectRouteOutcomeResolved {
		t.Fatalf("expected explicit directory query to resolve through facade, got %q (%s)", explicit.Kind, explicit.Error)
	}
	if explicit.Route.Kind != GatherContextDirectRouteDirectory {
		t.Fatalf("explicit.Route.Kind = %q, want %q", explicit.Route.Kind, GatherContextDirectRouteDirectory)
	}
	if got := explicit.Route.RawEntries(); len(got) != 1 || got[0] != "config" {
		t.Fatalf("unexpected explicit route entries: %+v", got)
	}

	if got := PlanGatherContextDirectRoute(execCtx, "Makefile", GatherContextDirectRoutePolicy{}); got.Kind != GatherContextDirectRouteOutcomeNone {
		t.Fatal("expected bare no-extension file to require explicit implicit-file policy")
	}
	if got := PlanGatherContextDirectRoute(execCtx, "sample.go", GatherContextDirectRoutePolicy{}); got.Kind != GatherContextDirectRouteOutcomeNone {
		t.Fatal("expected bare extension file to require explicit implicit-file policy")
	}

	sample := PlanGatherContextDirectRoute(execCtx, "sample.go", GatherContextDirectRoutePolicy{AllowImplicitBareFile: true})
	if sample.Kind != GatherContextDirectRouteOutcomeResolved {
		t.Fatalf("expected bare existing file to resolve when implicit policy is enabled, got %q (%s)", sample.Kind, sample.Error)
	}
	if sample.Route.Kind != GatherContextDirectRouteRead {
		t.Fatalf("sample.Route.Kind = %q, want %q", sample.Route.Kind, GatherContextDirectRouteRead)
	}

	for _, query := range []string{"main.ex", "main.exs", "types.pyi"} {
		resolved := PlanGatherContextDirectRoute(execCtx, query, GatherContextDirectRoutePolicy{AllowImplicitBareFile: true})
		if resolved.Kind != GatherContextDirectRouteOutcomeResolved {
			t.Fatalf("expected %q to resolve when implicit policy is enabled, got %q (%s)", query, resolved.Kind, resolved.Error)
		}
		if resolved.Route.Kind != GatherContextDirectRouteRead {
			t.Fatalf("%s route kind = %q, want %q", query, resolved.Route.Kind, GatherContextDirectRouteRead)
		}
	}

	implicit := PlanGatherContextDirectRoute(execCtx, "Makefile", GatherContextDirectRoutePolicy{AllowImplicitBareFile: true})
	if implicit.Kind != GatherContextDirectRouteOutcomeResolved {
		t.Fatalf("expected bare no-extension file to resolve when implicit policy is enabled, got %q (%s)", implicit.Kind, implicit.Error)
	}
	if implicit.Route.Kind != GatherContextDirectRouteRead {
		t.Fatalf("implicit.Route.Kind = %q, want %q", implicit.Route.Kind, GatherContextDirectRouteRead)
	}
	if got := implicit.Route.RawEntries(); len(got) != 1 || got[0] != "Makefile" {
		t.Fatalf("unexpected implicit route entries: %+v", got)
	}
}

func TestPlanGatherContextDirectRoute_ExplicitDirectoryPreservesFileFilter(t *testing.T) {
	root := t.TempDir()
	pkgDir := filepath.Join(root, "pkg")
	if err := os.MkdirAll(pkgDir, 0o755); err != nil {
		t.Fatal(err)
	}

	execCtx := tools.ExecutionContext{
		ProjectMapRootPath: root,
		InvocationCWD:      root,
	}

	for _, query := range []string{"./pkg/", pkgDir + string(os.PathSeparator)} {
		outcome := PlanGatherContextDirectRoute(execCtx, query, GatherContextDirectRoutePolicy{FileFilter: "go"})
		if outcome.Kind != GatherContextDirectRouteOutcomeResolved {
			t.Fatalf("expected explicit directory %q to resolve, got %q (%s)", query, outcome.Kind, outcome.Error)
		}
		if outcome.Route.Kind != GatherContextDirectRouteDirectory {
			t.Fatalf("%s route kind = %q, want %q", query, outcome.Route.Kind, GatherContextDirectRouteDirectory)
		}
		if len(outcome.Route.targets) != 1 {
			t.Fatalf("%s target count = %d, want 1", query, len(outcome.Route.targets))
		}
		if got := outcome.Route.targets[0].FileFilter; got != "go" {
			t.Fatalf("%s directory FileFilter = %q, want %q", query, got, "go")
		}
	}
}

func TestPlanGatherContextDirectRoute_SlashDelimitedPackageAndFileContracts(t *testing.T) {
	root := t.TempDir()
	for _, dir := range []string{
		filepath.Join(root, "internal", "agent"),
		filepath.Join(root, "pkg", "errors"),
	} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, "internal", "agent", "agent.go"), []byte("package agent\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "pkg", "errors.go"), []byte("package pkg\nconst sentinel = true\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	execCtx := tools.ExecutionContext{
		ProjectMapRootPath: root,
		InvocationCWD:      root,
	}

	for _, query := range []string{"pkg/errors", filepath.Join("internal", "agent")} {
		outcome := PlanGatherContextDirectRoute(execCtx, query, GatherContextDirectRoutePolicy{})
		if outcome.Kind != GatherContextDirectRouteOutcomeNone {
			t.Fatalf("expected %q to stay on search route, got %q (%s)", query, outcome.Kind, outcome.Error)
		}
	}

	for _, query := range []string{filepath.Join("internal", "agent") + string(os.PathSeparator), "." + string(os.PathSeparator) + filepath.Join("internal", "agent")} {
		outcome := PlanGatherContextDirectRoute(execCtx, query, GatherContextDirectRoutePolicy{})
		if outcome.Kind != GatherContextDirectRouteOutcomeResolved {
			t.Fatalf("expected %q to resolve as explicit directory query, got %q (%s)", query, outcome.Kind, outcome.Error)
		}
		if outcome.Route.Kind != GatherContextDirectRouteDirectory {
			t.Fatalf("%q route kind = %q, want %q", query, outcome.Route.Kind, GatherContextDirectRouteDirectory)
		}
		if got := outcome.Route.RawEntries(); len(got) != 1 || got[0] != filepath.Join("internal", "agent") {
			t.Fatalf("unexpected directory route entries for %q: %+v", query, got)
		}
	}

	for _, query := range []string{filepath.Join("pkg", "errors.go"), filepath.Join("pkg", "errors.go:1-10")} {
		outcome := PlanGatherContextDirectRoute(execCtx, query, GatherContextDirectRoutePolicy{})
		if outcome.Kind != GatherContextDirectRouteOutcomeResolved {
			t.Fatalf("expected %q to resolve as direct file query, got %q (%s)", query, outcome.Kind, outcome.Error)
		}
		if outcome.Route.Kind != GatherContextDirectRouteRead {
			t.Fatalf("%q route kind = %q, want %q", query, outcome.Route.Kind, GatherContextDirectRouteRead)
		}
	}
}

func TestPlanGatherContextDirectRoute_ClassifiesMissingDirectQueries(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	execCtx := tools.ExecutionContext{
		ProjectMapRootPath: root,
		InvocationCWD:      root,
	}

	missingPath := PlanGatherContextDirectRoute(execCtx, "./missing.go", GatherContextDirectRoutePolicy{})
	if missingPath.Kind != GatherContextDirectRouteOutcomeError {
		t.Fatalf("missingPath.Kind = %q, want %q", missingPath.Kind, GatherContextDirectRouteOutcomeError)
	}
	if !strings.Contains(missingPath.Error, "direct path not found") {
		t.Fatalf("expected missing path error, got: %q", missingPath.Error)
	}

	dottedPkg := PlanGatherContextDirectRoute(execCtx, "pkg.Func", GatherContextDirectRoutePolicy{AllowImplicitBareFile: true})
	if dottedPkg.Kind != GatherContextDirectRouteOutcomeNone {
		t.Fatalf("dottedPkg.Kind = %q, want %q", dottedPkg.Kind, GatherContextDirectRouteOutcomeNone)
	}

	dottedMethod := PlanGatherContextDirectRoute(execCtx, "Builder.Build", GatherContextDirectRoutePolicy{AllowImplicitBareFile: true})
	if dottedMethod.Kind != GatherContextDirectRouteOutcomeNone {
		t.Fatalf("dottedMethod.Kind = %q, want %q", dottedMethod.Kind, GatherContextDirectRouteOutcomeNone)
	}

	dottedModule := PlanGatherContextDirectRoute(execCtx, "MyModule.Version", GatherContextDirectRoutePolicy{AllowImplicitBareFile: true})
	if dottedModule.Kind != GatherContextDirectRouteOutcomeNone {
		t.Fatalf("dottedModule.Kind = %q, want %q", dottedModule.Kind, GatherContextDirectRouteOutcomeNone)
	}

	slashLiteral := PlanGatherContextDirectRoute(execCtx, "pkg/errors", GatherContextDirectRoutePolicy{AllowImplicitBareFile: true})
	if slashLiteral.Kind != GatherContextDirectRouteOutcomeNone {
		t.Fatalf("slashLiteral.Kind = %q, want %q", slashLiteral.Kind, GatherContextDirectRouteOutcomeNone)
	}

	importLiteral := PlanGatherContextDirectRoute(execCtx, "github.com/foo/bar", GatherContextDirectRoutePolicy{AllowImplicitBareFile: true})
	if importLiteral.Kind != GatherContextDirectRouteOutcomeNone {
		t.Fatalf("importLiteral.Kind = %q, want %q", importLiteral.Kind, GatherContextDirectRouteOutcomeNone)
	}

	mixedDirectAndSymbol := PlanGatherContextDirectRoute(execCtx, "go.mod,pkg.Func", GatherContextDirectRoutePolicy{AllowImplicitBareFile: true})
	if mixedDirectAndSymbol.Kind != GatherContextDirectRouteOutcomeNone {
		t.Fatalf("mixedDirectAndSymbol.Kind = %q, want %q", mixedDirectAndSymbol.Kind, GatherContextDirectRouteOutcomeNone)
	}

	missingBatch := PlanGatherContextDirectRoute(execCtx, "go.mod,missing.go", GatherContextDirectRoutePolicy{AllowImplicitBareFile: true})
	if missingBatch.Kind != GatherContextDirectRouteOutcomeError {
		t.Fatalf("missingBatch.Kind = %q, want %q", missingBatch.Kind, GatherContextDirectRouteOutcomeError)
	}
	if !strings.Contains(missingBatch.Error, "direct path not found") {
		t.Fatalf("expected missing batch error, got: %q", missingBatch.Error)
	}

	symbol := PlanGatherContextDirectRoute(execCtx, "Builder", GatherContextDirectRoutePolicy{AllowImplicitBareFile: true})
	if symbol.Kind != GatherContextDirectRouteOutcomeNone {
		t.Fatalf("symbol.Kind = %q, want %q", symbol.Kind, GatherContextDirectRouteOutcomeNone)
	}
}

func TestPlanGatherContextDirectRoute_ExplicitParentRelativeWithinRepo(t *testing.T) {
	root := t.TempDir()
	subdir := filepath.Join(root, "pkg")
	if err := os.MkdirAll(filepath.Join(subdir, "nested"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# root\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	execCtx := tools.ExecutionContext{
		ProjectMapRootPath: root,
		InvocationCWD:      subdir,
	}
	policy := GatherContextDirectRoutePolicy{FileFilter: "go"}

	read := PlanGatherContextDirectRoute(execCtx, "../README.md", policy)
	if read.Kind != GatherContextDirectRouteOutcomeResolved {
		t.Fatalf("expected parent-relative file to resolve, got %q (%s)", read.Kind, read.Error)
	}
	if read.Route.Kind != GatherContextDirectRouteRead {
		t.Fatalf("read.Route.Kind = %q, want %q", read.Route.Kind, GatherContextDirectRouteRead)
	}
	if got := read.Route.RawEntries(); len(got) != 1 || got[0] != filepath.Join("..", "README.md") {
		t.Fatalf("unexpected parent-relative file entries: %+v", got)
	}

	dir := PlanGatherContextDirectRoute(execCtx, "../pkg/", policy)
	if dir.Kind != GatherContextDirectRouteOutcomeResolved {
		t.Fatalf("expected parent-relative directory to resolve, got %q (%s)", dir.Kind, dir.Error)
	}
	if dir.Route.Kind != GatherContextDirectRouteDirectory {
		t.Fatalf("dir.Route.Kind = %q, want %q", dir.Route.Kind, GatherContextDirectRouteDirectory)
	}
	if got := dir.Route.RawEntries(); len(got) != 1 || got[0] != filepath.Join("..", "pkg") {
		t.Fatalf("unexpected parent-relative directory entries: %+v", got)
	}

	outside := PlanGatherContextDirectRoute(execCtx, "../../outside.txt", policy)
	if outside.Kind != GatherContextDirectRouteOutcomeError {
		t.Fatalf("expected escaping parent-relative path to error, got %q (%s)", outside.Kind, outside.Error)
	}
}

func TestPlanGatherContextDirectRoute_ExplicitDirectWithScopedPolicyUsesInvocationCWD(t *testing.T) {
	root := t.TempDir()
	subdir := filepath.Join(root, "pkg")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# root\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	execCtx := tools.ExecutionContext{
		ProjectMapRootPath: root,
		InvocationCWD:      subdir,
	}
	policy := GatherContextDirectRoutePolicy{FileFilter: "go"}

	missing := PlanGatherContextDirectRoute(execCtx, "./README.md", policy)
	if missing.Kind != GatherContextDirectRouteOutcomeError {
		t.Fatalf("expected explicit relative query to stay in invocation cwd, got %q (%s)", missing.Kind, missing.Error)
	}
	if !strings.Contains(missing.Error, "direct path not found: ./README.md") {
		t.Fatalf("expected explicit relative miss to stay direct, got %q", missing.Error)
	}

	if err := os.WriteFile(filepath.Join(subdir, "README.md"), []byte("# cwd\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	resolved := PlanGatherContextDirectRoute(execCtx, "./README.md", policy)
	if resolved.Kind != GatherContextDirectRouteOutcomeResolved {
		t.Fatalf("expected explicit relative query with stale filter to resolve directly, got %q (%s)", resolved.Kind, resolved.Error)
	}
	if got := resolved.Route.RawEntries(); len(got) != 1 || got[0] != "README.md" {
		t.Fatalf("unexpected explicit relative route entries: %+v", got)
	}
}

func TestPlanGatherContextDirectRoute_StrongExplicitPathIntentRemainsDirect(t *testing.T) {
	root := t.TempDir()
	subdir := filepath.Join(root, "cmd")
	if err := os.MkdirAll(filepath.Join(root, "foo"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(subdir, "child"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "foo", "bar.go"), []byte("package foo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subdir, "child", "local.go"), []byte("package child\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	execCtx := tools.ExecutionContext{
		ProjectMapRootPath: root,
		InvocationCWD:      subdir,
	}

	tests := []struct {
		query string
		want  string
		kind  GatherContextDirectRouteOutcomeKind
	}{
		{query: "./child/local.go", want: filepath.Join("child", "local.go"), kind: GatherContextDirectRouteOutcomeResolved},
		{query: "../foo/bar.go", want: filepath.Join("..", "foo", "bar.go"), kind: GatherContextDirectRouteOutcomeResolved},
		{query: filepath.Join(root, "foo", "bar.go"), want: filepath.Join(root, "foo", "bar.go"), kind: GatherContextDirectRouteOutcomeResolved},
	}

	for _, tt := range tests {
		outcome := PlanGatherContextDirectRoute(execCtx, tt.query, GatherContextDirectRoutePolicy{})
		if outcome.Kind != tt.kind {
			t.Fatalf("%q outcome kind = %q, want %q", tt.query, outcome.Kind, tt.kind)
		}
		if tt.kind == GatherContextDirectRouteOutcomeResolved {
			if outcome.Route.Kind != GatherContextDirectRouteRead {
				t.Fatalf("%q route kind = %q, want %q", tt.query, outcome.Route.Kind, GatherContextDirectRouteRead)
			}
			if got := outcome.Route.RawEntries(); len(got) != 1 || got[0] != tt.want {
				t.Fatalf("%q raw entries = %+v, want [%q]", tt.query, got, tt.want)
			}
			continue
		}
		if !strings.Contains(outcome.Error, "direct path not found") && outcome.Kind == GatherContextDirectRouteOutcomeError {
			t.Fatalf("%q expected direct error, got %q", tt.query, outcome.Error)
		}
	}
}
