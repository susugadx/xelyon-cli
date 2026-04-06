package file

import (
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/locator"
	"github.com/susugadx/xelyon-cli/internal/testutil"
	"github.com/susugadx/xelyon-cli/internal/tools"
	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

func TestReadFileTool_Targets(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "test.go", "package main\n\nfunc main() {\n\tprintln(\"hello\")\n}\n")

	// Registryに登録
	reg := locator.NewRegistry()
	id := reg.Register(locator.Location{
		FilePath: tmpDir + "/test.go",
		Line:     3,
		EndLine:  5,
	})

	if id != "[L1]" {
		t.Fatalf("expected [L1], got %s", id)
	}

	tool := &ReadFileTool{}
	execCtx := tools.ExecutionContext{
		LocatorRegistry: reg,
	}

	result, _, err := tool.Run(execCtx, map[string]string{
		"targets": "[L1]",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "func main()") {
		t.Errorf("expected file content with func main(), got:\n%s", result)
	}
}

func TestReadFileTool_TargetsCompactSingleLineUsesEnclosingBlock(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	filePath := tmpDir + "/compact.go"
	testutil.CreateTempFile(t, tmpDir, "compact.go", strings.Join([]string{
		"package main",
		"",
		"func alpha() {",
		"\tprintln(\"alpha\")",
		"}",
		"",
		"func beta() {",
		"\tprintln(\"beta\")",
		"\tprintln(\"gamma\")",
		"}",
	}, "\n"))

	reg := locator.NewRegistry()
	reg.Register(locator.Location{FilePath: filePath, Line: 8})

	tool := &ReadFileTool{}
	result, _, err := tool.Run(tools.ExecutionContext{LocatorRegistry: reg}, map[string]string{
		"targets": "[L1]",
		"detail":  "compact",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "📄 File: "+filePath+":7-10 [L2]") {
		t.Fatalf("expected expanded header with actual range, got:\n%s", result)
	}
	if !strings.Contains(result, "7: func beta() {") || !strings.Contains(result, "10: }") {
		t.Fatalf("expected enclosing block content, got:\n%s", result)
	}
	if strings.Contains(result, "3: func alpha() {") {
		t.Fatalf("compact locator read should not include sibling blocks, got:\n%s", result)
	}
}

func TestReadFileTool_TargetsCompactRespectsExistingLocatorRange(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	filePath := tmpDir + "/range.go"
	testutil.CreateTempFile(t, tmpDir, "range.go", strings.Join([]string{
		"package main",
		"",
		"func target() {",
		"\tprintln(\"one\")",
		"\tprintln(\"two\")",
		"}",
	}, "\n"))

	reg := locator.NewRegistry()
	reg.Register(locator.Location{FilePath: filePath, Line: 3, EndLine: 6})

	tool := &ReadFileTool{}
	result, _, err := tool.Run(tools.ExecutionContext{LocatorRegistry: reg}, map[string]string{
		"targets": "[L1]",
		"detail":  "compact",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "📄 File: "+filePath+":3-6 [L1]") {
		t.Fatalf("expected existing locator range to be preserved, got:\n%s", result)
	}
	if !strings.Contains(result, "3: func target() {") || !strings.Contains(result, "6: }") {
		t.Fatalf("expected preserved range content, got:\n%s", result)
	}
}

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

func TestReadFileTool_TargetsCompactFallsBackToWindow(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	filePath := tmpDir + "/notes.txt"
	lines := make([]string, 120)
	for i := range lines {
		lines[i] = "line" + strconv.Itoa(i+1)
	}
	testutil.CreateTempFile(t, tmpDir, "notes.txt", strings.Join(lines, "\n"))

	reg := locator.NewRegistry()
	reg.Register(locator.Location{FilePath: filePath, Line: 50})

	tool := &ReadFileTool{}
	result, _, err := tool.Run(tools.ExecutionContext{LocatorRegistry: reg}, map[string]string{
		"targets": "[L1]",
		"detail":  "compact",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "📄 File: "+filePath+":45-100 [L2]") {
		t.Fatalf("expected fallback window header, got:\n%s", result)
	}
	if !strings.Contains(result, "45: line45") || !strings.Contains(result, "100: line100") {
		t.Fatalf("expected fallback window content, got:\n%s", result)
	}
	if strings.Contains(result, "44: line44") {
		t.Fatalf("fallback window should start at line 45, got:\n%s", result)
	}
}

func TestReadFileTool_TargetsSingleLineFullReadsWholeFile(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	filePath := tmpDir + "/full.txt"
	lines := make([]string, 120)
	for i := range lines {
		lines[i] = "line" + strconv.Itoa(i+1)
	}
	testutil.CreateTempFile(t, tmpDir, "full.txt", strings.Join(lines, "\n"))

	reg := locator.NewRegistry()
	reg.Register(locator.Location{FilePath: filePath, Line: 50})

	tool := &ReadFileTool{}
	result, _, err := tool.Run(tools.ExecutionContext{LocatorRegistry: reg}, map[string]string{
		"targets": "[L1]",
		"detail":  "full",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "📄 File: "+filePath) {
		t.Fatalf("expected whole-file header, got:\n%s", result)
	}
	if !strings.Contains(result, "1: line1") || !strings.Contains(result, "120: line120") {
		t.Fatalf("detail=full should read the whole file for single-line locator, got:\n%s", result)
	}
	if strings.Contains(result, filePath+":45-100") {
		t.Fatalf("detail=full should not fall back to a locator window, got:\n%s", result)
	}
}

func TestReadFileTool_TargetsSingleLineOutlineReadsWholeFileOutline(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	filePath := tmpDir + "/outline.txt"
	lines := make([]string, 120)
	for i := range lines {
		lines[i] = "line" + strconv.Itoa(i+1)
	}
	testutil.CreateTempFile(t, tmpDir, "outline.txt", strings.Join(lines, "\n"))

	reg := locator.NewRegistry()
	reg.Register(locator.Location{FilePath: filePath, Line: 50})

	tool := &ReadFileTool{}
	result, _, err := tool.Run(tools.ExecutionContext{LocatorRegistry: reg}, map[string]string{
		"targets": "[L1]",
		"detail":  "outline",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "lines total") {
		t.Fatalf("detail=outline should force whole-file outline for single-line locator, got:\n%s", result)
	}
	if strings.Contains(result, "60: line60") {
		t.Fatalf("outline output should omit middle lines, got:\n%s", result)
	}
	if strings.Contains(result, filePath+":45-100") {
		t.Fatalf("detail=outline should not fall back to a locator window, got:\n%s", result)
	}
}

func TestReadFileTool_TargetsRangeDetailFullReadsWholeFile(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	filePath := tmpDir + "/range-full.txt"
	lines := make([]string, 120)
	for i := range lines {
		lines[i] = "line" + strconv.Itoa(i+1)
	}
	testutil.CreateTempFile(t, tmpDir, "range-full.txt", strings.Join(lines, "\n"))

	reg := locator.NewRegistry()
	reg.Register(locator.Location{FilePath: filePath, Line: 10, EndLine: 20})
	reg.Register(locator.Location{FilePath: filePath, Line: 60, EndLine: 70})

	tool := &ReadFileTool{}
	result, _, err := tool.Run(tools.ExecutionContext{LocatorRegistry: reg}, map[string]string{
		"targets": "[L1,L2]",
		"detail":  "full",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count := strings.Count(result, "📄 File: "); count != 1 {
		t.Fatalf("expected whole-file detail dedupe for range locators, got %d headers:\n%s", count, result)
	}
	if strings.Contains(result, filePath+":10-20") || strings.Contains(result, filePath+":60-70") {
		t.Fatalf("detail=full should override locator ranges to whole-file read, got:\n%s", result)
	}
	if !strings.Contains(result, "1: line1") || !strings.Contains(result, "120: line120") {
		t.Fatalf("expected whole-file content, got:\n%s", result)
	}
}

func TestReadFileTool_TargetsSingleLineRangeAutoPreservesExactRead(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	filePath := tmpDir + "/single-line.txt"
	lines := make([]string, 120)
	for i := range lines {
		lines[i] = "line" + strconv.Itoa(i+1)
	}
	testutil.CreateTempFile(t, tmpDir, "single-line.txt", strings.Join(lines, "\n"))

	reg := locator.NewRegistry()
	reg.Register(locator.Location{FilePath: filePath, Line: 50, EndLine: 50})

	tool := &ReadFileTool{}
	result, _, err := tool.Run(tools.ExecutionContext{LocatorRegistry: reg}, map[string]string{
		"targets": "[L1]",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "📄 File: "+filePath+":50-50 [L1]") {
		t.Fatalf("expected exact single-line locator range, got:\n%s", result)
	}
	if !strings.Contains(result, "50: line50") {
		t.Fatalf("expected exact target line, got:\n%s", result)
	}
	if strings.Contains(result, "49: line49") || strings.Contains(result, "51: line51") {
		t.Fatalf("detail=auto should preserve exact single-line locator span, got:\n%s", result)
	}
}

func TestReadFileTool_TargetsCompactFileLevelLocatorFallsBackToWholeFile(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	filePath := tmpDir + "/file-level.txt"
	lines := make([]string, 80)
	for i := range lines {
		lines[i] = "line" + strconv.Itoa(i+1)
	}
	testutil.CreateTempFile(t, tmpDir, "file-level.txt", strings.Join(lines, "\n"))

	reg := locator.NewRegistry()
	reg.Register(locator.Location{FilePath: filePath})

	tool := &ReadFileTool{}
	result, _, err := tool.Run(tools.ExecutionContext{LocatorRegistry: reg}, map[string]string{
		"targets": "[L1]",
		"detail":  "compact",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(result, `Error: detail="compact"`) {
		t.Fatalf("file-level locator compact should fall back instead of erroring, got:\n%s", result)
	}
	if !strings.Contains(result, "📄 File: "+filePath+" [L1]") {
		t.Fatalf("expected whole-file locator header, got:\n%s", result)
	}
	if !strings.Contains(result, "1: line1") || !strings.Contains(result, "80: line80") {
		t.Fatalf("expected whole-file fallback content, got:\n%s", result)
	}
}

func TestReadFileTool_TargetsCompactFileLevelLocatorLargeSingleLineFallsBackToOutline(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	filePath := tmpDir + "/manifest.min.js"
	testutil.CreateTempFile(t, tmpDir, "manifest.min.js", strings.Repeat("x", LargeFileThreshold+1024))

	reg := locator.NewRegistry()
	reg.Register(locator.Location{FilePath: filePath})

	tool := &ReadFileTool{}
	result, _, err := tool.Run(tools.ExecutionContext{LocatorRegistry: reg}, map[string]string{
		"targets": "[L1]",
		"detail":  "compact",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(result, "token too long") || strings.Contains(result, "Error reading file:") {
		t.Fatalf("file-level locator compact should not fail on large single-line files, got:\n%s", result)
	}
	if !strings.Contains(result, "📄 File: "+filePath+" [L1]") {
		t.Fatalf("expected whole-file locator header, got:\n%s", result)
	}
	if !strings.Contains(result, "lines total") || !strings.Contains(result, "...") {
		t.Fatalf("expected safe outline fallback, got:\n%s", result)
	}
	if len(result) > 10000 {
		t.Fatalf("file-level locator compact fallback should stay bounded, got %d bytes", len(result))
	}
}

func TestReadFileTool_TargetsWholeFileDetailDedupesSameFile(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	filePath := tmpDir + "/dedupe-full.txt"
	lines := make([]string, 120)
	for i := range lines {
		lines[i] = "line" + strconv.Itoa(i+1)
	}
	testutil.CreateTempFile(t, tmpDir, "dedupe-full.txt", strings.Join(lines, "\n"))

	reg := locator.NewRegistry()
	reg.Register(locator.Location{FilePath: filePath, Line: 10})
	reg.Register(locator.Location{FilePath: filePath, Line: 90})

	tool := &ReadFileTool{}
	result, _, err := tool.Run(tools.ExecutionContext{LocatorRegistry: reg}, map[string]string{
		"targets": "[L1,L2]",
		"detail":  "full",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count := strings.Count(result, "📄 File: "); count != 1 {
		t.Fatalf("expected deduped whole-file detail read, got %d headers:\n%s", count, result)
	}
	if !strings.Contains(result, "1: line1") || !strings.Contains(result, "120: line120") {
		t.Fatalf("expected whole-file content, got:\n%s", result)
	}
}

func TestReadFileTool_TargetsWholeFileDetailDedupesBeforeMaxReadValidation(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	filePath := tmpDir + "/dedupe-limit.txt"
	lines := make([]string, 120)
	for i := range lines {
		lines[i] = "line" + strconv.Itoa(i+1)
	}
	testutil.CreateTempFile(t, tmpDir, "dedupe-limit.txt", strings.Join(lines, "\n"))

	reg := locator.NewRegistry()
	targetIDs := make([]string, 0, MaxReadFilesPaths+1)
	for i := 0; i < MaxReadFilesPaths+1; i++ {
		id := reg.Register(locator.Location{FilePath: filePath, Line: i + 1})
		targetIDs = append(targetIDs, id)
	}

	tool := &ReadFileTool{}
	result, _, err := tool.Run(tools.ExecutionContext{LocatorRegistry: reg}, map[string]string{
		"targets": "[" + strings.Join(targetIDs, ",") + "]",
		"detail":  "full",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(result, "Error: too many paths") {
		t.Fatalf("deduped locator full read should not fail max-read validation, got:\n%s", result)
	}
	if count := strings.Count(result, "📄 File: "); count != 1 {
		t.Fatalf("expected one deduped whole-file result, got %d headers:\n%s", count, result)
	}
}

func TestReadFileTool_TargetsInvalidID(t *testing.T) {
	reg := locator.NewRegistry()

	tool := &ReadFileTool{}
	execCtx := tools.ExecutionContext{
		LocatorRegistry: reg,
	}

	result, _, err := tool.Run(execCtx, map[string]string{
		"targets": "[L99]",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(result, "Error: no valid locator IDs found") {
		t.Errorf("expected error for invalid targets, got: %s", result)
	}
}

func TestReadFileTool_LocatorIDInOutput(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "locator_test.txt", "line1\nline2\nline3\n")

	reg := locator.NewRegistry()

	result := ExecuteReadFilesWithLocator(
		common.DefaultOutput(),
		nil, nil,
		[]string{tmpDir + "/locator_test.txt"},
		DefaultFullLines,
		reg,
	)

	if !strings.Contains(result, "[L1]") {
		t.Errorf("expected Locator ID in output, got:\n%s", result)
	}
}

func TestReadFileTool_LocatorIDNilRegistry(t *testing.T) {
	setupTestMocks(t)
	defer withPermissiveValidatePath(t)()

	tmpDir := t.TempDir()
	testutil.CreateTempFile(t, tmpDir, "no_locator.txt", "line1\nline2\n")

	// nil registry → IDなし
	result := ExecuteReadFilesWithRuntime(
		common.DefaultOutput(),
		nil, nil,
		[]string{tmpDir + "/no_locator.txt"},
		DefaultFullLines,
	)

	if strings.Contains(result, "[L") {
		t.Errorf("expected no Locator ID with nil registry, got:\n%s", result)
	}
}
