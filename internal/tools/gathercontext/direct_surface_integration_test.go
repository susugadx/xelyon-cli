package gathercontext

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/locator"
)

func TestGatherContext_DirectReadSurfaceContracts(t *testing.T) {
	root := t.TempDir()
	withGatherContextWorkingDir(t, root)

	writeGatherContextFiles(t, map[string]string{
		filepath.Join(root, "sample.txt"): strings.Join([]string{
			"line1",
			"line2",
			"line3",
			"line4",
			"line5",
		}, "\n"),
		filepath.Join(root, "nested", "child.txt"): "child\n",
	})

	reg := locator.NewRegistry()
	reg.Register(locator.Location{
		FilePath: "sample.txt",
		Line:     3,
		EndLine:  3,
		Name:     "target line",
	})

	execCtx := newGatherContextExecCtx(root, withGatherContextLocatorRegistry(reg))

	t.Run("locator query uses compact read", func(t *testing.T) {
		result, change := runGatherContext(t, execCtx, map[string]string{"query": "[L1]"})
		if change != nil {
			t.Fatalf("expected no file change, got %+v", change)
		}
		assertGatherContextContainsAll(t, result, "Route: Direct read", "📄 File: sample.txt:3-3 [", "3: line3")
	})

	t.Run("explicit range query preserves exact range", func(t *testing.T) {
		result, _ := runGatherContext(t, execCtx, map[string]string{"query": "sample.txt:3-4"})
		if !strings.Contains(result, "📄 File: /") && !strings.Contains(result, "📄 File: sample.txt:3-4") {
			t.Fatalf("expected range read header, got:\n%s", result)
		}
		assertGatherContextContainsAll(t, result, "3: line3", "4: line4")
		assertGatherContextExcludesAll(t, result, "2: line2", "5: line5")
	})

	t.Run("direct file query uses whole-file auto read", func(t *testing.T) {
		result, _ := runGatherContext(t, execCtx, map[string]string{"query": "sample.txt"})
		if !strings.Contains(result, "📄 File: /") && !strings.Contains(result, "📄 File: sample.txt") {
			t.Fatalf("expected file read header, got:\n%s", result)
		}
		assertGatherContextContainsAll(t, result, "1: line1", "5: line5")
	})

	t.Run("comma separated direct file query reads all files without search fallback", func(t *testing.T) {
		result, _ := runGatherContext(t, execCtx, map[string]string{"query": "sample.txt,nested/child.txt"})
		assertGatherContextContainsAll(t, result, "Route: Direct read", "📄 File: sample.txt", "📄 File: nested/child.txt")
		assertGatherContextExcludesAll(t, result, "No matches found")
	})

	t.Run("explicit direct directory query uses list_dir renderer", func(t *testing.T) {
		result, _ := runGatherContext(t, execCtx, map[string]string{"query": "nested/"})
		assertGatherContextContainsAll(t, result, "Route: Directory listing", "📂", "summary: depth=1", "files: child.txt")
	})

	t.Run("missing explicit path returns direct error instead of search fallback", func(t *testing.T) {
		result, _ := runGatherContext(t, execCtx, map[string]string{"query": "./missing.txt"})
		if !strings.HasPrefix(strings.TrimSpace(result), "Error:") {
			t.Fatalf("expected direct error to preserve Error prefix, got:\n%s", result)
		}
		assertGatherContextContainsAll(t, result, "Route: Direct query", "Error: direct path not found: ./missing.txt")
		assertGatherContextExcludesAll(t, result, "No matches found")
	})

	t.Run("mixed direct and dotted symbol query does not partial-direct", func(t *testing.T) {
		result, _ := runGatherContext(t, execCtx, map[string]string{"query": "sample.txt,pkg.Func"})
		assertGatherContextExcludesAll(t, result, "Route: Direct read", "Error: direct path not found")
	})

	t.Run("missing file in direct batch returns direct error", func(t *testing.T) {
		result, _ := runGatherContext(t, execCtx, map[string]string{"query": "sample.txt,missing.txt"})
		assertGatherContextContainsAll(t, result, "Route: Direct query", "Error: direct path not found: missing.txt")
		assertGatherContextExcludesAll(t, result, "Route: Auto search", "No matches found")
	})
}

func TestGatherContext_SlashDelimitedExplicitDirectSurfaceContracts(t *testing.T) {
	root := t.TempDir()
	withGatherContextWorkingDir(t, root)

	writeGatherContextFiles(t, map[string]string{
		filepath.Join(root, "internal", "agent", "agent.go"): "package agent\n",
		filepath.Join(root, "pkg", "errors.go"):              "package pkg\nconst sentinel = true\n",
	})

	execCtx := newGatherContextExecCtx(root)

	t.Run("slash directory marker stays on directory route", func(t *testing.T) {
		result, _ := runGatherContext(t, execCtx, map[string]string{"query": filepath.Join("internal", "agent") + string(filepath.Separator)})
		assertGatherContextContainsAll(t, result, "Route: Directory listing", "agent.go")
		assertGatherContextExcludesAll(t, result, "Route: Direct read", "No matches found")
	})

	t.Run("explicit relative slash directory stays on directory route", func(t *testing.T) {
		query := "." + string(filepath.Separator) + filepath.Join("internal", "agent")
		result, _ := runGatherContext(t, execCtx, map[string]string{"query": query})
		assertGatherContextContainsAll(t, result, "Route: Directory listing", "agent.go")
		assertGatherContextExcludesAll(t, result, "Route: Direct read", "No matches found")
	})

	t.Run("slash file path with extension stays on direct read", func(t *testing.T) {
		result, _ := runGatherContext(t, execCtx, map[string]string{"query": filepath.Join("pkg", "errors.go")})
		assertGatherContextContainsAll(t, result, "Route: Direct read", "📄 File: pkg/errors.go", "const sentinel = true")
		assertGatherContextExcludesAll(t, result, "Route: Directory listing", "No matches found")
	})
}

func TestGatherContext_EditToolSurfaceControlsFullExactRead(t *testing.T) {
	root := t.TempDir()
	withGatherContextWorkingDir(t, root)

	lines := make([]string, 2205)
	for i := range lines {
		lines[i] = fmt.Sprintf("line%d", i+1)
	}
	writeGatherContextFiles(t, map[string]string{
		filepath.Join(root, "big.go"): strings.Join(lines, "\n"),
	})

	reg := locator.NewRegistry()
	reg.Register(locator.Location{
		FilePath: "big.go",
		Line:     100,
		EndLine:  100,
		Name:     "line100",
	})

	applyPatchExecCtx := newGatherContextExecCtx(
		root,
		withGatherContextLocatorRegistry(reg),
		withGatherContextModel("openai", "gpt-5.4"),
	)
	legacyExecCtx := newGatherContextExecCtx(
		root,
		withGatherContextLocatorRegistry(reg),
		withGatherContextModel("kimi", "kimi-k2.6"),
	)

	t.Run("single exact file prefers full read", func(t *testing.T) {
		result, _ := runGatherContext(t, applyPatchExecCtx, map[string]string{"query": "./big.go"})
		assertGatherContextContainsAll(t, result, "Route: Direct read", "1: line1", "1001: line1001", "2205: line2205")
		assertGatherContextExcludesAll(t, result, "lines total")
	})

	t.Run("legacy provider keeps auto read", func(t *testing.T) {
		result, _ := runGatherContext(t, legacyExecCtx, map[string]string{"query": "./big.go"})
		assertGatherContextContainsAll(t, result, "Route: Direct read", "lines total")
		assertGatherContextExcludesAll(t, result, "1001: line1001")
	})

	t.Run("locator read stays compact", func(t *testing.T) {
		result, _ := runGatherContext(t, applyPatchExecCtx, map[string]string{"query": "[L1]"})
		assertGatherContextContainsAll(t, result, "Route: Direct read", "📄 File: big.go:100-100 [")
		assertGatherContextExcludesAll(t, result, "2205: line2205")
	})

	t.Run("explicit range read stays exact", func(t *testing.T) {
		result, _ := runGatherContext(t, applyPatchExecCtx, map[string]string{"query": "./big.go:100-101"})
		assertGatherContextContainsAll(t, result, "100: line100", "101: line101")
		assertGatherContextExcludesAll(t, result, "99: line99", "102: line102", "2205: line2205", "lines total")
	})
}
