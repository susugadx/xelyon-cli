package readtool

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/locator"
	"github.com/susugadx/xelyon-cli/internal/testutil"
	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestReadFileTool_TargetsCompactLocatorRelativePathUsesProjectRootInSubdir(t *testing.T) {
	setupTestMocks(t)

	root := t.TempDir()
	subdir := filepath.Join(root, "sub")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	testutil.CreateTempFile(t, root, "pkg/run.go", strings.Join([]string{
		"package main",
		"",
		"func run() {",
		"\tprintln(\"root-target\")",
		"}",
	}, "\n"))
	testutil.CreateTempFile(t, subdir, "pkg/run.go", strings.Join([]string{
		"package main",
		"",
		"func run() {",
		"\tprintln(\"wrong-subdir\")",
		"}",
	}, "\n"))

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	if err := os.Chdir(subdir); err != nil {
		t.Fatalf("failed to chdir to subdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origDir)
	})

	reg := locator.NewRegistry()
	reg.Register(locator.Location{FilePath: "pkg/run.go", Line: 3})

	execCtx := tools.ExecutionContext{
		LocatorRegistry:    reg,
		ProjectMapRootPath: root,
		InvocationCWD:      subdir,
		Stdin:              strings.NewReader(""),
		Stdout:             io.Discard,
		Stderr:             io.Discard,
	}

	tool := &ReadFileTool{}
	result, _, err := tool.Run(execCtx, map[string]string{
		"targets": "[L1]",
		"detail":  "compact",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(result, "wrong-subdir") {
		t.Fatalf("expected locator relative path to preserve project-root semantics, got:\n%s", result)
	}
	if !strings.Contains(result, "root-target") {
		t.Fatalf("expected compact locator read to use project-root file, got:\n%s", result)
	}
	if !strings.Contains(result, "📄 File: pkg/run.go:3-5") {
		t.Fatalf("expected display path to stay project-relative, got:\n%s", result)
	}
}

func TestReadFileTool_TargetsCompactLocatorResolvedPathUsesSubdirHit(t *testing.T) {
	setupTestMocks(t)

	root := t.TempDir()
	subdir := filepath.Join(root, "sub")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	testutil.CreateTempFile(t, root, "target.go", strings.Join([]string{
		"package main",
		"",
		"func run() {",
		"\tprintln(\"root-target\")",
		"}",
	}, "\n"))
	subdirTarget := testutil.CreateTempFile(t, subdir, "target.go", strings.Join([]string{
		"package main",
		"",
		"func run() {",
		"\tprintln(\"subdir-target\")",
		"}",
	}, "\n"))

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	if err := os.Chdir(subdir); err != nil {
		t.Fatalf("failed to chdir to subdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origDir)
	})

	reg := locator.NewRegistry()
	reg.Register(locator.Location{FilePath: "target.go", ResolvedPath: subdirTarget, Line: 3})

	execCtx := tools.ExecutionContext{
		LocatorRegistry:    reg,
		ProjectMapRootPath: root,
		InvocationCWD:      subdir,
		Stdin:              strings.NewReader(""),
		Stdout:             io.Discard,
		Stderr:             io.Discard,
	}

	tool := &ReadFileTool{}
	result, _, err := tool.Run(execCtx, map[string]string{
		"targets": "[L1]",
		"detail":  "compact",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(result, "root-target") {
		t.Fatalf("expected resolved path to override project-root fallback, got:\n%s", result)
	}
	if !strings.Contains(result, "subdir-target") {
		t.Fatalf("expected compact locator read to use resolved subdir file, got:\n%s", result)
	}
	if !strings.Contains(result, "📄 File: target.go:3-5") {
		t.Fatalf("expected display path to stay locator-relative, got:\n%s", result)
	}
}

func TestReadFileTool_TargetsLocatorResolvedPathOutsideWorkspaceIsRejected(t *testing.T) {
	setupTestMocks(t)

	root := t.TempDir()
	subdir := filepath.Join(root, "sub")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	outsideDir := t.TempDir()
	outsideFile := testutil.CreateTempFile(t, outsideDir, "outside.go", strings.Join([]string{
		"package main",
		"",
		"func outside() {",
		"\tprintln(\"outside\")",
		"}",
	}, "\n"))

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	if err := os.Chdir(subdir); err != nil {
		t.Fatalf("failed to chdir to subdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origDir)
	})

	reg := locator.NewRegistry()
	reg.Register(locator.Location{FilePath: "outside.go", ResolvedPath: outsideFile, Line: 3})

	execCtx := tools.ExecutionContext{
		LocatorRegistry:    reg,
		ProjectMapRootPath: root,
		InvocationCWD:      subdir,
		Stdin:              strings.NewReader(""),
		Stdout:             io.Discard,
		Stderr:             io.Discard,
	}

	tool := &ReadFileTool{}
	result, _, err := tool.Run(execCtx, map[string]string{
		"targets": "[L1]",
		"detail":  "compact",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "Error: path escape attempt detected:") {
		t.Fatalf("expected outside resolved path to be rejected, got:\n%s", result)
	}
	if strings.Contains(result, "println(\"outside\")") {
		t.Fatalf("expected outside file content to stay unread, got:\n%s", result)
	}
}

func TestReadFileTool_PathsRelativeReadReemitsResolvedPathForFollowUpTargets(t *testing.T) {
	setupTestMocks(t)

	root := t.TempDir()
	subdir := filepath.Join(root, "sub")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	testutil.CreateTempFile(t, root, "target.go", strings.Join([]string{
		"package main",
		"",
		"func run() {",
		"\tprintln(\"root-target\")",
		"}",
	}, "\n"))
	subdirTarget := testutil.CreateTempFile(t, subdir, "target.go", strings.Join([]string{
		"package main",
		"",
		"func run() {",
		"\tprintln(\"subdir-target\")",
		"}",
	}, "\n"))

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	if err := os.Chdir(subdir); err != nil {
		t.Fatalf("failed to chdir to subdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origDir)
	})

	reg := locator.NewRegistry()
	execCtx := tools.ExecutionContext{
		LocatorRegistry:    reg,
		ProjectMapRootPath: root,
		InvocationCWD:      subdir,
		Stdin:              strings.NewReader(""),
		Stdout:             io.Discard,
		Stderr:             io.Discard,
	}

	tool := &ReadFileTool{}
	first, _, err := tool.Run(execCtx, map[string]string{
		"paths": `["target.go"]`,
	})
	if err != nil {
		t.Fatalf("unexpected error on initial read: %v", err)
	}
	if !strings.Contains(first, "subdir-target") {
		t.Fatalf("expected initial relative read to use subdir file, got:\n%s", first)
	}

	loc, ok := reg.Resolve("[L1]")
	if !ok {
		t.Fatal("expected locator [L1] from initial direct read")
	}
	if loc.ResolvedPath != subdirTarget {
		t.Fatalf("expected emitted locator to preserve resolved path %s, got %+v", subdirTarget, loc)
	}

	second, _, err := tool.Run(execCtx, map[string]string{
		"targets": "[L1]",
		"detail":  "compact",
	})
	if err != nil {
		t.Fatalf("unexpected error on follow-up read: %v", err)
	}
	if strings.Contains(second, "root-target") {
		t.Fatalf("expected follow-up locator read to stay on subdir file, got:\n%s", second)
	}
	if !strings.Contains(second, "subdir-target") {
		t.Fatalf("expected follow-up locator read to use subdir file, got:\n%s", second)
	}
}

func TestReadFileTool_PathsRelativeFollowUpTargetsSurviveDeeperCWD(t *testing.T) {
	setupTestMocks(t)

	root := t.TempDir()
	deeper := filepath.Join(root, "sub", "deep")
	if err := os.MkdirAll(deeper, 0o755); err != nil {
		t.Fatalf("failed to create deeper cwd: %v", err)
	}

	testutil.CreateTempFile(t, root, "foo.go", strings.Join([]string{
		"package main",
		"",
		"func run() {",
		"\tprintln(\"root-target\")",
		"}",
	}, "\n"))

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("failed to chdir to root: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origDir)
	})

	reg := locator.NewRegistry()
	execCtx := tools.ExecutionContext{
		LocatorRegistry: reg,
		Stdin:           strings.NewReader(""),
		Stdout:          io.Discard,
		Stderr:          io.Discard,
		InvocationCWD:   root,
	}

	tool := &ReadFileTool{}
	first, _, err := tool.Run(execCtx, map[string]string{
		"paths": `["foo.go"]`,
	})
	if err != nil {
		t.Fatalf("unexpected error on initial read: %v", err)
	}
	if !strings.Contains(first, "root-target") {
		t.Fatalf("expected initial read to use root file, got:\n%s", first)
	}

	if err := os.Chdir(deeper); err != nil {
		t.Fatalf("failed to chdir to deeper cwd: %v", err)
	}
	execCtx.InvocationCWD = deeper

	second, _, err := tool.Run(execCtx, map[string]string{
		"targets": "[L1]",
	})
	if err != nil {
		t.Fatalf("unexpected error on follow-up read: %v", err)
	}
	if strings.Contains(second, "Error: path escape attempt detected:") {
		t.Fatalf("expected follow-up read to survive deeper cwd, got:\n%s", second)
	}
	if !strings.Contains(second, "root-target") {
		t.Fatalf("expected follow-up read to use original root file, got:\n%s", second)
	}
}

func TestReadFileTool_PathsRelativeReadFollowUpTargetsWorkFromSymlinkWorkspace(t *testing.T) {
	setupTestMocks(t)

	realRoot := t.TempDir()
	realWorkspace := filepath.Join(realRoot, "workspace")
	if err := os.MkdirAll(realWorkspace, 0o755); err != nil {
		t.Fatalf("failed to create real workspace: %v", err)
	}
	linkRoot := filepath.Join(t.TempDir(), "workspace-link")
	if err := os.Symlink(realWorkspace, linkRoot); err != nil {
		t.Skipf("symlink unsupported: %v", err)
	}

	testutil.CreateTempFile(t, realWorkspace, "foo.go", strings.Join([]string{
		"package main",
		"",
		"func run() {",
		"\tprintln(\"symlink-workspace\")",
		"}",
	}, "\n"))

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	if err := os.Chdir(linkRoot); err != nil {
		t.Fatalf("failed to chdir to symlink workspace: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origDir)
	})

	reg := locator.NewRegistry()
	execCtx := tools.ExecutionContext{
		LocatorRegistry:    reg,
		ProjectMapRootPath: linkRoot,
		InvocationCWD:      linkRoot,
		Stdin:              strings.NewReader(""),
		Stdout:             io.Discard,
		Stderr:             io.Discard,
	}

	tool := &ReadFileTool{}
	first, _, err := tool.Run(execCtx, map[string]string{
		"paths": `["foo.go"]`,
	})
	if err != nil {
		t.Fatalf("unexpected error on initial symlink read: %v", err)
	}
	if !strings.Contains(first, "symlink-workspace") {
		t.Fatalf("expected initial read to succeed through symlink workspace, got:\n%s", first)
	}

	loc, ok := reg.Resolve("[L1]")
	if !ok {
		t.Fatal("expected locator [L1] from initial symlink read")
	}
	if loc.ResolvedPath == "" {
		t.Fatalf("expected symlink read locator to keep resolved path, got %+v", loc)
	}

	second, _, err := tool.Run(execCtx, map[string]string{
		"targets": "[L1]",
	})
	if err != nil {
		t.Fatalf("unexpected error on follow-up symlink read: %v", err)
	}
	if strings.Contains(second, "Error: path escape attempt detected:") {
		t.Fatalf("expected symlink follow-up locator read to succeed, got:\n%s", second)
	}
	if !strings.Contains(second, "symlink-workspace") {
		t.Fatalf("expected follow-up read to use real file content, got:\n%s", second)
	}
}

func TestReadFileTool_TargetsCompactLocatorRelativePathFallsBackToInvocationCWDWithoutProjectRoot(t *testing.T) {
	setupTestMocks(t)

	root := t.TempDir()
	subdir := filepath.Join(root, "sub")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	testutil.CreateTempFile(t, subdir, "pkg/run.go", strings.Join([]string{
		"package main",
		"",
		"func run() {",
		"\tprintln(\"cwd-fallback\")",
		"}",
	}, "\n"))

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	if err := os.Chdir(subdir); err != nil {
		t.Fatalf("failed to chdir to subdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origDir)
	})

	reg := locator.NewRegistry()
	reg.Register(locator.Location{FilePath: "pkg/run.go", Line: 3})

	execCtx := tools.ExecutionContext{
		LocatorRegistry: reg,
		InvocationCWD:   subdir,
		Stdin:           strings.NewReader(""),
		Stdout:          io.Discard,
		Stderr:          io.Discard,
	}

	tool := &ReadFileTool{}
	result, _, err := tool.Run(execCtx, map[string]string{
		"targets": "[L1]",
		"detail":  "compact",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "cwd-fallback") {
		t.Fatalf("expected locator relative path to fall back to invocation cwd when project root is unavailable, got:\n%s", result)
	}
	if !strings.Contains(result, "📄 File: pkg/run.go:3-5") {
		t.Fatalf("expected display path to stay project-relative, got:\n%s", result)
	}
}

func TestReadFileTool_TargetsCompactAbsoluteLocatorPathUsesAbsoluteFile(t *testing.T) {
	setupTestMocks(t)

	root := t.TempDir()
	subdir := filepath.Join(root, "sub")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatalf("failed to create subdir: %v", err)
	}

	filePath := testutil.CreateTempFile(t, root, "pkg/run.go", strings.Join([]string{
		"package main",
		"",
		"func run() {",
		"\tprintln(\"absolute-target\")",
		"}",
	}, "\n"))

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get cwd: %v", err)
	}
	if err := os.Chdir(subdir); err != nil {
		t.Fatalf("failed to chdir to subdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origDir)
	})

	reg := locator.NewRegistry()
	reg.Register(locator.Location{FilePath: filePath, Line: 3})

	execCtx := tools.ExecutionContext{
		LocatorRegistry:    reg,
		ProjectMapRootPath: root,
		InvocationCWD:      subdir,
		Stdin:              strings.NewReader(""),
		Stdout:             io.Discard,
		Stderr:             io.Discard,
	}

	tool := &ReadFileTool{}
	result, _, err := tool.Run(execCtx, map[string]string{
		"targets": "[L1]",
		"detail":  "compact",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "absolute-target") {
		t.Fatalf("expected absolute locator path to resolve directly, got:\n%s", result)
	}
	if !strings.Contains(result, "📄 File: "+filePath+":3-5") {
		t.Fatalf("expected display path to preserve absolute locator path, got:\n%s", result)
	}
}
