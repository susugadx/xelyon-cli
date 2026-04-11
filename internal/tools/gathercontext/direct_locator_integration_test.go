package gathercontext

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/locator"
)

func TestGatherContext_LocatorFollowUpSurface(t *testing.T) {
	root := t.TempDir()
	withGatherContextWorkingDir(t, root)

	writeGatherContextFiles(t, map[string]string{
		filepath.Join(root, "sample.txt"): strings.Join([]string{
			"line1",
			"line2",
			"line3",
		}, "\n"),
	})

	reg := locator.NewRegistry()
	reg.Register(locator.Location{
		FilePath: "sample.txt",
		Line:     2,
		EndLine:  2,
		Name:     "selected line",
	})
	reg.Register(locator.Location{
		FilePath: "sample.txt",
		Line:     3,
		EndLine:  3,
		Name:     "next line",
	})

	execCtx := newGatherContextExecCtx(root, withGatherContextLocatorRegistry(reg))

	t.Run("bare", func(t *testing.T) {
		result, _ := runGatherContext(t, execCtx, map[string]string{"query": "L1"})
		assertGatherContextContainsAll(t, result, "Route: Direct read", "📄 File: sample.txt:2-2 [", "2: line2")
		assertGatherContextExcludesAll(t, result, "Search / Discovery")
	})

	t.Run("mixed lowercase list", func(t *testing.T) {
		result, _ := runGatherContext(t, execCtx, map[string]string{"query": "[l1, L2]"})
		assertGatherContextContainsAll(t, result, "Route: Direct read", "2: line2", "3: line3")
		assertGatherContextExcludesAll(t, result, "Search / Discovery")
	})

	t.Run("bare list with unresolved id stays off locator route", func(t *testing.T) {
		result, _ := runGatherContext(t, execCtx, map[string]string{"query": "L1,L999"})
		assertGatherContextExcludesAll(t, result, "Route: Direct read", "📄 File: sample.txt:2-2 [")
	})
}

func TestGatherContext_LocatorReadUsesResolvedPathFollowUp(t *testing.T) {
	root := t.TempDir()
	subdir := filepath.Join(root, "pkg")

	writeGatherContextFiles(t, map[string]string{
		filepath.Join(root, "target.go"):   "package main\nconst selected = \"root\"\n",
		filepath.Join(subdir, "target.go"): "package pkg\nconst selected = \"subdir\"\n",
	})

	reg := locator.NewRegistry()
	reg.Register(locator.Location{
		FilePath:     "target.go",
		ResolvedPath: filepath.Join(subdir, "target.go"),
		Line:         2,
		EndLine:      2,
		Name:         "selected",
	})

	result, _ := runGatherContext(t, newGatherContextExecCtx(
		root,
		withGatherContextLocatorRegistry(reg),
		withGatherContextInvocationCWD(subdir),
	), map[string]string{"query": "[L1]"})
	assertGatherContextContainsAll(t, result, `2: const selected = "subdir"`)
	assertGatherContextExcludesAll(t, result, `"root"`)
}
