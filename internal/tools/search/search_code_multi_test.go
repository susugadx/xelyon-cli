package search

import (
	"path/filepath"
	"testing"
)

func TestBuildFormattedPatternExecution_TrimsLineRangeHint(t *testing.T) {
	execution := buildFormattedPatternExecution(2, singlePatternExecution{
		Pattern: "Run",
		Output:  "result body" + lineRangeHint,
	})
	if execution.Index != 2 {
		t.Fatalf("expected index 2, got %d", execution.Index)
	}
	if execution.Output != "result body" {
		t.Fatalf("expected output without hint suffix, got %q", execution.Output)
	}
}

func TestCollectOrderedPatternExecutions_ByIndex(t *testing.T) {
	ch := make(chan formattedPatternExecution, 3)
	ch <- formattedPatternExecution{Index: 2, singlePatternExecution: singlePatternExecution{Pattern: "c"}}
	ch <- formattedPatternExecution{Index: 0, singlePatternExecution: singlePatternExecution{Pattern: "a"}}
	ch <- formattedPatternExecution{Index: 1, singlePatternExecution: singlePatternExecution{Pattern: "b"}}

	collected := collectOrderedPatternExecutions(3, ch)
	if len(collected) != 3 {
		t.Fatalf("expected 3 collected executions, got %d", len(collected))
	}
	if collected[0].Pattern != "a" || collected[1].Pattern != "b" || collected[2].Pattern != "c" {
		t.Fatalf("unexpected ordering: %+v", collected)
	}
}

func TestPrepareMultiPatternCacheWrite(t *testing.T) {
	dir := setupMultiLangDir(t, map[string]string{
		"run.go": "package example\n\nfunc Run() {}\n",
	})
	opts := SearchOptions{Pattern: "Run,helper(", Path: dir, InvocationCWD: dir}
	contexts := newSinglePatternExecutionContexts([]string{"Run", "helper("}, opts)
	runPath := filepath.Join(dir, "run.go")

	collected := []formattedPatternExecution{
		{Index: 0, singlePatternExecution: singlePatternExecution{Pattern: "Run", AffectedFiles: []string{runPath}}},
		{Index: 1, singlePatternExecution: singlePatternExecution{Pattern: "helper(", AffectedFiles: []string{runPath}}},
	}

	got := prepareMultiPatternCacheWrite(contexts, opts, collected)

	if got.PatternKey != buildMultiCacheKeyFromContexts(contexts) {
		t.Fatalf("unexpected pattern key: got=%q want=%q", got.PatternKey, buildMultiCacheKeyFromContexts(contexts))
	}
	if got.CacheKey != buildMultiSearchCacheKeyFromContexts(opts, contexts) {
		t.Fatalf("unexpected cache key: got=%q want=%q", got.CacheKey, buildMultiSearchCacheKeyFromContexts(opts, contexts))
	}
	if len(got.AffectedFiles) != 1 || got.AffectedFiles[0] != runPath {
		t.Fatalf("unexpected affected files: %v", got.AffectedFiles)
	}
}
