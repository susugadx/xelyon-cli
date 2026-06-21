package directquery

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestPlan_Policy(t *testing.T) {
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

	explicit := Plan(execCtx, "config/", Policy{})
	if explicit.Kind != OutcomeResolved {
		t.Fatalf("expected explicit directory query to resolve through facade, got %q (%s)", explicit.Kind, explicit.Error)
	}
	if explicit.Route.Kind != RouteDirectory {
		t.Fatalf("explicit.Route.Kind = %q, want %q", explicit.Route.Kind, RouteDirectory)
	}
	if got := explicit.Route.RawEntries(); len(got) != 1 || got[0] != "config" {
		t.Fatalf("unexpected explicit route entries: %+v", got)
	}

	if got := Plan(execCtx, "Makefile", Policy{}); got.Kind != OutcomeNone {
		t.Fatal("expected bare no-extension file to require explicit implicit-file policy")
	}
	if got := Plan(execCtx, "sample.go", Policy{}); got.Kind != OutcomeNone {
		t.Fatal("expected bare extension file to require explicit implicit-file policy")
	}

	sample := Plan(execCtx, "sample.go", Policy{AllowImplicitBareFile: true})
	if sample.Kind != OutcomeResolved {
		t.Fatalf("expected bare existing file to resolve when implicit policy is enabled, got %q (%s)", sample.Kind, sample.Error)
	}
	if sample.Route.Kind != RouteRead {
		t.Fatalf("sample.Route.Kind = %q, want %q", sample.Route.Kind, RouteRead)
	}

	for _, query := range []string{"main.ex", "main.exs", "types.pyi"} {
		resolved := Plan(execCtx, query, Policy{AllowImplicitBareFile: true})
		if resolved.Kind != OutcomeResolved {
			t.Fatalf("expected %q to resolve when implicit policy is enabled, got %q (%s)", query, resolved.Kind, resolved.Error)
		}
		if resolved.Route.Kind != RouteRead {
			t.Fatalf("%s route kind = %q, want %q", query, resolved.Route.Kind, RouteRead)
		}
	}

	implicit := Plan(execCtx, "Makefile", Policy{AllowImplicitBareFile: true})
	if implicit.Kind != OutcomeResolved {
		t.Fatalf("expected bare no-extension file to resolve when implicit policy is enabled, got %q (%s)", implicit.Kind, implicit.Error)
	}
	if implicit.Route.Kind != RouteRead {
		t.Fatalf("implicit.Route.Kind = %q, want %q", implicit.Route.Kind, RouteRead)
	}
	if got := implicit.Route.RawEntries(); len(got) != 1 || got[0] != "Makefile" {
		t.Fatalf("unexpected implicit route entries: %+v", got)
	}
}

func TestPlan_ExplicitDirectoryPreservesFileFilter(t *testing.T) {
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
		outcome := Plan(execCtx, query, Policy{FileFilter: "go"})
		if outcome.Kind != OutcomeResolved {
			t.Fatalf("expected explicit directory %q to resolve, got %q (%s)", query, outcome.Kind, outcome.Error)
		}
		if outcome.Route.Kind != RouteDirectory {
			t.Fatalf("%s route kind = %q, want %q", query, outcome.Route.Kind, RouteDirectory)
		}
		if len(outcome.Route.targets) != 1 {
			t.Fatalf("%s target count = %d, want 1", query, len(outcome.Route.targets))
		}
		if got := outcome.Route.targets[0].FileFilter; got != "go" {
			t.Fatalf("%s directory FileFilter = %q, want %q", query, got, "go")
		}
	}
}

func TestPlan_SlashDelimitedPackageAndFileContracts(t *testing.T) {
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
		outcome := Plan(execCtx, query, Policy{})
		if outcome.Kind != OutcomeNone {
			t.Fatalf("expected %q to stay on search route, got %q (%s)", query, outcome.Kind, outcome.Error)
		}
	}

	for _, query := range []string{filepath.Join("internal", "agent") + string(os.PathSeparator), "." + string(os.PathSeparator) + filepath.Join("internal", "agent")} {
		outcome := Plan(execCtx, query, Policy{})
		if outcome.Kind != OutcomeResolved {
			t.Fatalf("expected %q to resolve as explicit directory query, got %q (%s)", query, outcome.Kind, outcome.Error)
		}
		if outcome.Route.Kind != RouteDirectory {
			t.Fatalf("%q route kind = %q, want %q", query, outcome.Route.Kind, RouteDirectory)
		}
		if got := outcome.Route.RawEntries(); len(got) != 1 || got[0] != filepath.Join("internal", "agent") {
			t.Fatalf("unexpected directory route entries for %q: %+v", query, got)
		}
	}

	for _, query := range []string{filepath.Join("pkg", "errors.go"), filepath.Join("pkg", "errors.go:1-10")} {
		outcome := Plan(execCtx, query, Policy{})
		if outcome.Kind != OutcomeResolved {
			t.Fatalf("expected %q to resolve as direct file query, got %q (%s)", query, outcome.Kind, outcome.Error)
		}
		if outcome.Route.Kind != RouteRead {
			t.Fatalf("%q route kind = %q, want %q", query, outcome.Route.Kind, RouteRead)
		}
	}
}

func TestPlan_ClassifiesMissingDirectQueries(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/test\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	execCtx := tools.ExecutionContext{
		ProjectMapRootPath: root,
		InvocationCWD:      root,
	}

	missingPath := Plan(execCtx, "./missing.go", Policy{})
	if missingPath.Kind != OutcomeError {
		t.Fatalf("missingPath.Kind = %q, want %q", missingPath.Kind, OutcomeError)
	}
	if !strings.Contains(missingPath.Error, "direct path not found") {
		t.Fatalf("expected missing path error, got: %q", missingPath.Error)
	}

	dottedPkg := Plan(execCtx, "pkg.Func", Policy{AllowImplicitBareFile: true})
	if dottedPkg.Kind != OutcomeNone {
		t.Fatalf("dottedPkg.Kind = %q, want %q", dottedPkg.Kind, OutcomeNone)
	}

	dottedMethod := Plan(execCtx, "Builder.Build", Policy{AllowImplicitBareFile: true})
	if dottedMethod.Kind != OutcomeNone {
		t.Fatalf("dottedMethod.Kind = %q, want %q", dottedMethod.Kind, OutcomeNone)
	}

	dottedModule := Plan(execCtx, "MyModule.Version", Policy{AllowImplicitBareFile: true})
	if dottedModule.Kind != OutcomeNone {
		t.Fatalf("dottedModule.Kind = %q, want %q", dottedModule.Kind, OutcomeNone)
	}

	slashLiteral := Plan(execCtx, "pkg/errors", Policy{AllowImplicitBareFile: true})
	if slashLiteral.Kind != OutcomeNone {
		t.Fatalf("slashLiteral.Kind = %q, want %q", slashLiteral.Kind, OutcomeNone)
	}

	importLiteral := Plan(execCtx, "github.com/foo/bar", Policy{AllowImplicitBareFile: true})
	if importLiteral.Kind != OutcomeNone {
		t.Fatalf("importLiteral.Kind = %q, want %q", importLiteral.Kind, OutcomeNone)
	}

	mixedDirectAndSymbol := Plan(execCtx, "go.mod,pkg.Func", Policy{AllowImplicitBareFile: true})
	if mixedDirectAndSymbol.Kind != OutcomeNone {
		t.Fatalf("mixedDirectAndSymbol.Kind = %q, want %q", mixedDirectAndSymbol.Kind, OutcomeNone)
	}

	missingBatch := Plan(execCtx, "go.mod,missing.go", Policy{AllowImplicitBareFile: true})
	if missingBatch.Kind != OutcomeError {
		t.Fatalf("missingBatch.Kind = %q, want %q", missingBatch.Kind, OutcomeError)
	}
	if !strings.Contains(missingBatch.Error, "direct path not found") {
		t.Fatalf("expected missing batch error, got: %q", missingBatch.Error)
	}

	symbol := Plan(execCtx, "Builder", Policy{AllowImplicitBareFile: true})
	if symbol.Kind != OutcomeNone {
		t.Fatalf("symbol.Kind = %q, want %q", symbol.Kind, OutcomeNone)
	}
}

func TestPlan_ExplicitParentRelativeWithinRepo(t *testing.T) {
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
	policy := Policy{FileFilter: "go"}

	read := Plan(execCtx, "../README.md", policy)
	if read.Kind != OutcomeResolved {
		t.Fatalf("expected parent-relative file to resolve, got %q (%s)", read.Kind, read.Error)
	}
	if read.Route.Kind != RouteRead {
		t.Fatalf("read.Route.Kind = %q, want %q", read.Route.Kind, RouteRead)
	}
	if got := read.Route.RawEntries(); len(got) != 1 || got[0] != filepath.Join("..", "README.md") {
		t.Fatalf("unexpected parent-relative file entries: %+v", got)
	}

	dir := Plan(execCtx, "../pkg/", policy)
	if dir.Kind != OutcomeResolved {
		t.Fatalf("expected parent-relative directory to resolve, got %q (%s)", dir.Kind, dir.Error)
	}
	if dir.Route.Kind != RouteDirectory {
		t.Fatalf("dir.Route.Kind = %q, want %q", dir.Route.Kind, RouteDirectory)
	}
	if got := dir.Route.RawEntries(); len(got) != 1 || got[0] != filepath.Join("..", "pkg") {
		t.Fatalf("unexpected parent-relative directory entries: %+v", got)
	}

	outside := Plan(execCtx, "../../outside.txt", policy)
	if outside.Kind != OutcomeError {
		t.Fatalf("expected escaping parent-relative path to error, got %q (%s)", outside.Kind, outside.Error)
	}
}

func TestPlan_ExplicitDirectWithScopedPolicyUsesInvocationCWD(t *testing.T) {
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
	policy := Policy{FileFilter: "go"}

	missing := Plan(execCtx, "./README.md", policy)
	if missing.Kind != OutcomeError {
		t.Fatalf("expected explicit relative query to stay in invocation cwd, got %q (%s)", missing.Kind, missing.Error)
	}
	if !strings.Contains(missing.Error, "direct path not found: ./README.md") {
		t.Fatalf("expected explicit relative miss to stay direct, got %q", missing.Error)
	}

	if err := os.WriteFile(filepath.Join(subdir, "README.md"), []byte("# cwd\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	resolved := Plan(execCtx, "./README.md", policy)
	if resolved.Kind != OutcomeResolved {
		t.Fatalf("expected explicit relative query with stale filter to resolve directly, got %q (%s)", resolved.Kind, resolved.Error)
	}
	if got := resolved.Route.RawEntries(); len(got) != 1 || got[0] != "README.md" {
		t.Fatalf("unexpected explicit relative route entries: %+v", got)
	}
}

func TestPlan_StrongExplicitPathIntentRemainsDirect(t *testing.T) {
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
		kind  OutcomeKind
	}{
		{query: "./child/local.go", want: filepath.Join("child", "local.go"), kind: OutcomeResolved},
		{query: "../foo/bar.go", want: filepath.Join("..", "foo", "bar.go"), kind: OutcomeResolved},
		{query: filepath.Join(root, "foo", "bar.go"), want: filepath.Join(root, "foo", "bar.go"), kind: OutcomeResolved},
	}

	for _, tt := range tests {
		outcome := Plan(execCtx, tt.query, Policy{})
		if outcome.Kind != tt.kind {
			t.Fatalf("%q outcome kind = %q, want %q", tt.query, outcome.Kind, tt.kind)
		}
		if tt.kind == OutcomeResolved {
			if outcome.Route.Kind != RouteRead {
				t.Fatalf("%q route kind = %q, want %q", tt.query, outcome.Route.Kind, RouteRead)
			}
			if got := outcome.Route.RawEntries(); len(got) != 1 || got[0] != tt.want {
				t.Fatalf("%q raw entries = %+v, want [%q]", tt.query, got, tt.want)
			}
			continue
		}
		if !strings.Contains(outcome.Error, "direct path not found") && outcome.Kind == OutcomeError {
			t.Fatalf("%q expected direct error, got %q", tt.query, outcome.Error)
		}
	}
}
