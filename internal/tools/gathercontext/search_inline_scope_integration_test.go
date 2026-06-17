package gathercontext

import (
	"path/filepath"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/tools/common"
)

func TestGatherContext_NaturalLanguageInlineScopeUsesSearchRoute(t *testing.T) {
	if !common.IsRipgrepAvailable() {
		t.Skip("ripgrep not available")
	}
	common.ResetRipgrepAvailabilityForTest()
	t.Cleanup(common.ResetRipgrepAvailabilityForTest)

	root := t.TempDir()
	withGatherContextWorkingDir(t, root)

	writeGatherContextFiles(t, map[string]string{
		filepath.Join(root, "docs", "review.md"): "review harness docs\nreview command notes\n",
		filepath.Join(root, "README.md"):         "review harness outside docs\n",
	})

	result, _ := runGatherContext(t, newGatherContextExecCtx(root), map[string]string{
		"query": "docs or review harness or review-harness or review command in docs/",
	})

	assertGatherContextContainsAll(t, result, "Route: Auto search", "docs/review.md")
	assertGatherContextExcludesAll(t, result, "Route: Direct query", "direct path not found", "README.md")
}

func TestGatherContext_NaturalLanguageInlineScopeStripsSearchPrefix(t *testing.T) {
	if !common.IsRipgrepAvailable() {
		t.Skip("ripgrep not available")
	}
	common.ResetRipgrepAvailabilityForTest()
	t.Cleanup(common.ResetRipgrepAvailabilityForTest)

	root := t.TempDir()
	withGatherContextWorkingDir(t, root)

	writeGatherContextFiles(t, map[string]string{
		filepath.Join(root, "docs", "review.md"): "foo docs\n",
		filepath.Join(root, "README.md"):         "search for foo outside docs\n",
	})

	result, _ := runGatherContext(t, newGatherContextExecCtx(root), map[string]string{
		"query": "search for foo or quux in docs/",
	})

	assertGatherContextContainsAll(t, result, "Route: Auto search", "docs/review.md")
	assertGatherContextExcludesAll(t, result, "search for foo", "README.md")
}

func TestGatherContext_NaturalLanguageInlineScopePreservesQuotedOrPhrase(t *testing.T) {
	if !common.IsRipgrepAvailable() {
		t.Skip("ripgrep not available")
	}
	common.ResetRipgrepAvailabilityForTest()
	t.Cleanup(common.ResetRipgrepAvailabilityForTest)

	root := t.TempDir()
	withGatherContextWorkingDir(t, root)

	writeGatherContextFiles(t, map[string]string{
		filepath.Join(root, "docs", "review.md"): "foo or bar\n",
		filepath.Join(root, "docs", "noise.md"):  "foo only\nbar only\n",
		filepath.Join(root, "README.md"):         "foo or bar outside docs\n",
	})

	result, _ := runGatherContext(t, newGatherContextExecCtx(root), map[string]string{
		"query": `search for "foo or bar" in docs/`,
	})

	assertGatherContextContainsAll(t, result, "Route: Auto search", "docs/review.md")
	assertGatherContextExcludesAll(t, result, "docs/noise.md", "README.md", `"foo`, `bar"`)
}

func TestGatherContext_NaturalLanguageInlineScopeAcceptsGerundPrefix(t *testing.T) {
	if !common.IsRipgrepAvailable() {
		t.Skip("ripgrep not available")
	}
	common.ResetRipgrepAvailabilityForTest()
	t.Cleanup(common.ResetRipgrepAvailabilityForTest)

	root := t.TempDir()
	withGatherContextWorkingDir(t, root)

	writeGatherContextFiles(t, map[string]string{
		filepath.Join(root, "docs", "review.md"): "foo docs\n",
		filepath.Join(root, "README.md"):         "looking for foo outside docs\n",
	})

	result, _ := runGatherContext(t, newGatherContextExecCtx(root), map[string]string{
		"query": "looking for foo in docs/",
	})

	assertGatherContextContainsAll(t, result, "Route:", "docs/review.md")
	assertGatherContextExcludesAll(t, result, "Route: Direct query", "direct path not found", "README.md")
}

func TestGatherContext_NaturalLanguagePrepositionDoesNotBecomeInlineScope(t *testing.T) {
	if !common.IsRipgrepAvailable() {
		t.Skip("ripgrep not available")
	}
	common.ResetRipgrepAvailabilityForTest()
	t.Cleanup(common.ResetRipgrepAvailabilityForTest)

	root := t.TempDir()
	withGatherContextWorkingDir(t, root)

	writeGatherContextFiles(t, map[string]string{
		filepath.Join(root, "notes.md"): "timeout in handler\n",
	})

	result, _ := runGatherContext(t, newGatherContextExecCtx(root), map[string]string{
		"query": "search for timeout in handler",
	})

	assertGatherContextContainsAll(t, result, "Route:", "notes.md")
	assertGatherContextExcludesAll(t, result, "Route: Direct query", "direct path not found")
}

func TestGatherContext_NaturalLanguageInlineScopeAcceptsBareFileName(t *testing.T) {
	if !common.IsRipgrepAvailable() {
		t.Skip("ripgrep not available")
	}
	common.ResetRipgrepAvailabilityForTest()
	t.Cleanup(common.ResetRipgrepAvailabilityForTest)

	root := t.TempDir()
	withGatherContextWorkingDir(t, root)

	writeGatherContextFiles(t, map[string]string{
		filepath.Join(root, "handler.go"): "package main\n\nvar timeout = true\n",
		filepath.Join(root, "notes.md"):   "timeout in handler.go\n",
	})

	result, _ := runGatherContext(t, newGatherContextExecCtx(root), map[string]string{
		"query": "search for timeout in handler.go",
	})

	assertGatherContextContainsAll(t, result, "Route:", "handler.go")
	assertGatherContextExcludesAll(t, result, "No matches found", "notes.md")
}
