package directquery

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestResolveCandidateGatherContextReadRoute(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "README.md"), []byte("# doc\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	execCtx := tools.ExecutionContext{
		Stdout:             io.Discard,
		Stderr:             io.Discard,
		ProjectMapRootPath: root,
		InvocationCWD:      root,
	}

	input, ok := parseDirectQueryInput("README.md")
	if !ok {
		t.Fatal("parseDirectQueryInput(README.md) = false, want true")
	}
	outcome := resolveCandidateGatherContextReadRoute(execCtx, input)
	if outcome.Kind != OutcomeResolved {
		t.Fatalf("resolveCandidateGatherContextReadRoute() kind = %q, want resolved (%s)", outcome.Kind, outcome.Error)
	}
	if outcome.Route.Kind != RouteRead {
		t.Fatalf("route kind = %q, want %q", outcome.Route.Kind, RouteRead)
	}

	missingInput, ok := parseDirectQueryInput("missing.go")
	if !ok {
		t.Fatal("parseDirectQueryInput(missing.go) = false, want true")
	}
	missing := resolveCandidateGatherContextReadRoute(execCtx, missingInput)
	if missing.Kind != OutcomeNone {
		t.Fatalf("single missing candidate kind = %q, want none", missing.Kind)
	}

	batchInput, ok := parseDirectQueryInput("README.md,missing.go")
	if !ok {
		t.Fatal("parseDirectQueryInput(batch) = false, want true")
	}
	batch := resolveCandidateGatherContextReadRoute(execCtx, batchInput)
	if batch.Kind != OutcomeError {
		t.Fatalf("batch missing kind = %q, want error", batch.Kind)
	}
	if !strings.Contains(batch.Error, "direct path not found") {
		t.Fatalf("batch missing error = %q, want direct path error", batch.Error)
	}
}

func TestResolveCandidateGatherContextAnyRoute_ResolvesDirectoryAndRead(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "pkg", "service.go"), []byte("package pkg\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	execCtx := tools.ExecutionContext{
		Stdout:             io.Discard,
		Stderr:             io.Discard,
		ProjectMapRootPath: root,
		InvocationCWD:      root,
	}

	dirInput, ok := parseDirectQueryInput("pkg/")
	if !ok {
		t.Fatal("parseDirectQueryInput(pkg/) = false, want true")
	}
	dirOutcome := resolveCandidateGatherContextAnyRoute(execCtx, dirInput, Policy{FileFilter: "go"})
	if dirOutcome.Kind != OutcomeResolved {
		t.Fatalf("directory outcome kind = %q, want resolved (%s)", dirOutcome.Kind, dirOutcome.Error)
	}
	if dirOutcome.Route.Kind != RouteDirectory {
		t.Fatalf("directory route kind = %q, want directory", dirOutcome.Route.Kind)
	}
	if got := dirOutcome.Route.targets[0].FileFilter; got != "go" {
		t.Fatalf("directory FileFilter = %q, want %q", got, "go")
	}

	fileInput, ok := parseDirectQueryInput("pkg/service.go")
	if !ok {
		t.Fatal("parseDirectQueryInput(pkg/service.go) = false, want true")
	}
	fileOutcome := resolveRequiredCandidateGatherContextAnyRoute(execCtx, fileInput, Policy{})
	if fileOutcome.Kind != OutcomeResolved {
		t.Fatalf("required any-route kind = %q, want resolved (%s)", fileOutcome.Kind, fileOutcome.Error)
	}
	if fileOutcome.Route.Kind != RouteRead {
		t.Fatalf("file route kind = %q, want read", fileOutcome.Route.Kind)
	}
}

func TestExecuteGatherContextDirectRoute_DispatchesByRouteKind(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "pkg"), 0o755); err != nil {
		t.Fatal(err)
	}
	targetFile := filepath.Join(root, "pkg", "service.go")
	if err := os.WriteFile(targetFile, []byte("package pkg\nconst selected = true\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	execCtx := tools.ExecutionContext{
		Stdout:             io.Discard,
		Stderr:             io.Discard,
		ProjectMapRootPath: root,
		InvocationCWD:      root,
	}

	readResult := executeRoute(execCtx, Route{
		Kind: RouteRead,
		targets: []directQueryTarget{{
			RawEntry:     "pkg/service.go",
			FilePath:     filepath.Join("pkg", "service.go"),
			ResolvedPath: targetFile,
			AllowedRoots: []string{root},
			Kind:         directQueryTargetFile,
		}},
	}, "auto", 1)
	if !strings.Contains(readResult, "const selected = true") {
		t.Fatalf("read dispatch output missing file content, got:\n%s", readResult)
	}

	dirResult := executeRoute(execCtx, Route{
		Kind: RouteDirectory,
		targets: []directQueryTarget{{
			RawEntry:      "pkg",
			FilePath:      "pkg",
			ResolvedPath:  filepath.Join(root, "pkg"),
			AllowedRoots:  []string{root},
			WorkspaceRoot: root,
			FileFilter:    "go",
			Kind:          directQueryTargetDirectory,
		}},
	}, "", 1)
	if !strings.Contains(dirResult, "service.go") {
		t.Fatalf("directory dispatch output missing listing, got:\n%s", dirResult)
	}

	if got := executeRoute(execCtx, Route{Kind: RouteDirectory}, "", 1); got != "Error: path is not a directory" {
		t.Fatalf("empty directory route = %q, want error", got)
	}
	if got := executeRoute(execCtx, Route{}, "", 1); got != "" {
		t.Fatalf("unknown route kind output = %q, want empty string", got)
	}
}
