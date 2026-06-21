package directquery

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools"
)

func TestResolveDirectQueryTarget_PrefersInvocationCWD(t *testing.T) {
	root := t.TempDir()
	subdir := filepath.Join(root, "pkg")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatal(err)
	}

	rootTarget := filepath.Join(root, "target.go")
	subTarget := filepath.Join(subdir, "target.go")
	if err := os.WriteFile(rootTarget, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(subTarget, []byte("package pkg\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	targetInput, ok := parseDirectQueryEntryInput("target.go")
	if !ok {
		t.Fatal("expected direct query input to parse")
	}
	target, errResult := resolveDirectQueryTarget(tools.ExecutionContext{
		ProjectMapRootPath: root,
		InvocationCWD:      subdir,
	}, targetInput)
	if errResult != "" {
		t.Fatalf("expected direct query target to resolve, got %q", errResult)
	}
	if target.ResolvedPath != subTarget {
		t.Fatalf("target.ResolvedPath = %q, want %q", target.ResolvedPath, subTarget)
	}
}

func TestResolveDirectQueryTarget_ExplicitRelativeUsesCWDButAllowsInRepoParent(t *testing.T) {
	root := t.TempDir()
	subdir := filepath.Join(root, "pkg")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatal(err)
	}

	rootTarget := filepath.Join(root, "Makefile")
	if err := os.WriteFile(rootTarget, []byte("all:\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	input, ok := parseDirectQueryEntryInput("./Makefile")
	if !ok {
		t.Fatal("expected explicit relative query to parse")
	}
	if _, errResult := resolveDirectQueryTarget(tools.ExecutionContext{
		ProjectMapRootPath: root,
		InvocationCWD:      subdir,
	}, input); errResult == "" {
		t.Fatal("expected explicit relative query to stay within invocation cwd")
	}

	subTarget := filepath.Join(subdir, "Makefile")
	if err := os.WriteFile(subTarget, []byte("all:\n\tgo test ./...\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	target, errResult := resolveDirectQueryTarget(tools.ExecutionContext{
		ProjectMapRootPath: root,
		InvocationCWD:      subdir,
	}, input)
	if errResult != "" {
		t.Fatalf("expected explicit relative query to resolve once file exists in invocation cwd, got %q", errResult)
	}
	if target.ResolvedPath != subTarget {
		t.Fatalf("target.ResolvedPath = %q, want %q", target.ResolvedPath, subTarget)
	}

	parentInput, ok := parseDirectQueryEntryInput("../Makefile")
	if !ok {
		t.Fatal("expected parent-relative query to parse")
	}
	parentTarget, errResult := resolveDirectQueryTarget(tools.ExecutionContext{
		ProjectMapRootPath: root,
		InvocationCWD:      subdir,
	}, parentInput)
	if errResult != "" {
		t.Fatalf("expected in-repo parent-relative query to resolve, got %q", errResult)
	}
	if parentTarget.ResolvedPath != rootTarget {
		t.Fatalf("parentTarget.ResolvedPath = %q, want %q", parentTarget.ResolvedPath, rootTarget)
	}

	parentDirInput, ok := parseDirectQueryEntryInput("../pkg/")
	if !ok {
		t.Fatal("expected parent-relative directory query to parse")
	}
	parentDirTarget, errResult := resolveDirectQueryTarget(tools.ExecutionContext{
		ProjectMapRootPath: root,
		InvocationCWD:      subdir,
	}, parentDirInput)
	if errResult != "" {
		t.Fatalf("expected in-repo parent-relative directory query to resolve, got %q", errResult)
	}
	if parentDirTarget.Kind != directQueryTargetDirectory || parentDirTarget.ResolvedPath != subdir {
		t.Fatalf("unexpected parent-relative directory target: %+v", parentDirTarget)
	}

	outsideRoot := t.TempDir()
	outsideInput, ok := parseDirectQueryEntryInput("../../outside.txt")
	if !ok {
		t.Fatal("expected outside parent-relative query to parse")
	}
	if err := os.WriteFile(filepath.Join(outsideRoot, "outside.txt"), []byte("outside\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, errResult := resolveDirectQueryTarget(tools.ExecutionContext{
		ProjectMapRootPath: root,
		InvocationCWD:      subdir,
	}, outsideInput); errResult == "" {
		t.Fatal("expected parent-relative path escaping repo roots to be rejected")
	}
}
