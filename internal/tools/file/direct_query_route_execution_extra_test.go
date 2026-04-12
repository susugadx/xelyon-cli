package file

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
	if outcome.Kind != GatherContextDirectRouteOutcomeResolved {
		t.Fatalf("resolveCandidateGatherContextReadRoute() kind = %q, want resolved (%s)", outcome.Kind, outcome.Error)
	}
	if outcome.Route.Kind != GatherContextDirectRouteRead {
		t.Fatalf("route kind = %q, want %q", outcome.Route.Kind, GatherContextDirectRouteRead)
	}

	missingInput, ok := parseDirectQueryInput("missing.go")
	if !ok {
		t.Fatal("parseDirectQueryInput(missing.go) = false, want true")
	}
	missing := resolveCandidateGatherContextReadRoute(execCtx, missingInput)
	if missing.Kind != GatherContextDirectRouteOutcomeNone {
		t.Fatalf("single missing candidate kind = %q, want none", missing.Kind)
	}

	batchInput, ok := parseDirectQueryInput("README.md,missing.go")
	if !ok {
		t.Fatal("parseDirectQueryInput(batch) = false, want true")
	}
	batch := resolveCandidateGatherContextReadRoute(execCtx, batchInput)
	if batch.Kind != GatherContextDirectRouteOutcomeError {
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
	dirOutcome := resolveCandidateGatherContextAnyRoute(execCtx, dirInput, GatherContextDirectRoutePolicy{FileFilter: "go"})
	if dirOutcome.Kind != GatherContextDirectRouteOutcomeResolved {
		t.Fatalf("directory outcome kind = %q, want resolved (%s)", dirOutcome.Kind, dirOutcome.Error)
	}
	if dirOutcome.Route.Kind != GatherContextDirectRouteDirectory {
		t.Fatalf("directory route kind = %q, want directory", dirOutcome.Route.Kind)
	}
	if got := dirOutcome.Route.targets[0].FileFilter; got != "go" {
		t.Fatalf("directory FileFilter = %q, want %q", got, "go")
	}

	fileInput, ok := parseDirectQueryInput("pkg/service.go")
	if !ok {
		t.Fatal("parseDirectQueryInput(pkg/service.go) = false, want true")
	}
	fileOutcome := resolveRequiredCandidateGatherContextAnyRoute(execCtx, fileInput, GatherContextDirectRoutePolicy{})
	if fileOutcome.Kind != GatherContextDirectRouteOutcomeResolved {
		t.Fatalf("required any-route kind = %q, want resolved (%s)", fileOutcome.Kind, fileOutcome.Error)
	}
	if fileOutcome.Route.Kind != GatherContextDirectRouteRead {
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

	readResult := ExecuteGatherContextDirectRoute(execCtx, GatherContextDirectRoute{
		Kind: GatherContextDirectRouteRead,
		targets: []DirectQueryTarget{{
			RawEntry:     "pkg/service.go",
			FilePath:     filepath.Join("pkg", "service.go"),
			ResolvedPath: targetFile,
			AllowedRoots: []string{root},
			Kind:         DirectQueryTargetFile,
		}},
	}, "auto", 1)
	if !strings.Contains(readResult, "const selected = true") {
		t.Fatalf("read dispatch output missing file content, got:\n%s", readResult)
	}

	dirResult := ExecuteGatherContextDirectRoute(execCtx, GatherContextDirectRoute{
		Kind: GatherContextDirectRouteDirectory,
		targets: []DirectQueryTarget{{
			RawEntry:      "pkg",
			FilePath:      "pkg",
			ResolvedPath:  filepath.Join(root, "pkg"),
			AllowedRoots:  []string{root},
			WorkspaceRoot: root,
			FileFilter:    "go",
			Kind:          DirectQueryTargetDirectory,
		}},
	}, "", 1)
	if !strings.Contains(dirResult, "service.go") {
		t.Fatalf("directory dispatch output missing listing, got:\n%s", dirResult)
	}

	if got := ExecuteGatherContextDirectRoute(execCtx, GatherContextDirectRoute{Kind: GatherContextDirectRouteDirectory}, "", 1); got != "Error: path is not a directory" {
		t.Fatalf("empty directory route = %q, want error", got)
	}
	if got := ExecuteGatherContextDirectRoute(execCtx, GatherContextDirectRoute{}, "", 1); got != "" {
		t.Fatalf("unknown route kind output = %q, want empty string", got)
	}
}
