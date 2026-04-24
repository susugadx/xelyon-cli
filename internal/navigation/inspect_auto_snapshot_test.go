package navigation

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/repomap"
	"github.com/susugadx/xelyon-cli/internal/searchcache"
)

func TestResolveInspectSymbolAuto_UsesSnapshotCacheWhenProjectMapBecomesNil(t *testing.T) {
	dir := setupTestGoFiles(t, map[string]string{
		"cached.go": "package example\n\nfunc Run() {}\n",
	})

	opts := InspectSymbolAutoOptions{
		Budget: SummaryBudget,
		ProjectMap: &repomap.ProjectMap{
			RootPath: dir,
			Files: []*repomap.FileEntry{
				{
					Path: "cached.go",
					Symbols: []repomap.Symbol{
						{Name: "Run", Kind: "function", Line: 3, EndLine: 3, Signature: "func Run() {}", Exported: true},
					},
				},
			},
		},
		ProjectMapRootPath: dir,
		ProjectMapStateKey: "snapshot-cache-state",
	}

	result, _, status := ResolveInspectSymbolAuto("Run", filepath.Join(dir, "cached.go"), opts)
	if status != SymbolAutoSingle || result.Symbol == nil {
		t.Fatalf("expected initial snapshot-backed resolution, got status=%s result=%+v", status, result)
	}

	if err := os.WriteFile(filepath.Join(dir, "cached.go"), []byte("package example\n\nfunc (\n"), 0o644); err != nil {
		t.Fatalf("failed to break source file: %v", err)
	}

	opts.ProjectMap = nil
	result, _, status = ResolveInspectSymbolAuto("Run", filepath.Join(dir, "cached.go"), opts)
	if status != SymbolAutoSingle || result.Symbol == nil {
		t.Fatalf("expected cached snapshot-backed resolution after ProjectMap removal, got status=%s result=%+v", status, result)
	}
	if result.Symbol.File != "cached.go" {
		t.Fatalf("unexpected cached snapshot symbol: %+v", result.Symbol)
	}
}

func TestLoadGoSymbolSnapshot_ReusesCachedEntryBeforeRebuild(t *testing.T) {
	dir := t.TempDir()
	cacheKey := goSymbolSnapshotCacheKey(dir, "reuse-state")
	want := &goSymbolSnapshot{
		RootPath: dir,
		StateKey: "reuse-state",
		ByName: map[string][]goSymbolSnapshotEntry{
			"Run": {{Name: "Run", File: "cached.go", Line: 3}},
		},
	}
	storeGoSymbolSnapshot(cacheKey, want)
	t.Cleanup(clearGoSymbolSnapshotCache)

	got := loadGoSymbolSnapshot(GoSymbolRuntime{
		ProjectMap: &repomap.ProjectMap{
			RootPath: dir,
			Files: []*repomap.FileEntry{
				{
					Path: "other.go",
					Symbols: []repomap.Symbol{
						{Name: "Other", Kind: "function", Line: 5, Signature: "func Other()"},
					},
				},
			},
		},
		ProjectMapRootPath: dir,
		ProjectMapStateKey: "reuse-state",
	})
	if got != want {
		t.Fatalf("expected cached snapshot pointer reuse, got %+v want %+v", got, want)
	}
}

func TestGoSymbolSnapshotCacheClearedBySearchCacheHook(t *testing.T) {
	dir := t.TempDir()
	storeGoSymbolSnapshot(goSymbolSnapshotCacheKey(dir, "clear-state"), &goSymbolSnapshot{
		RootPath: dir,
		StateKey: "clear-state",
		ByName:   map[string][]goSymbolSnapshotEntry{"Run": {{Name: "Run"}}},
	})

	if got := lookupGoSymbolSnapshot(goSymbolSnapshotCacheKey(dir, "clear-state")); got == nil {
		t.Fatal("expected snapshot cache entry before clear")
	}

	searchcache.NotifySearchCacheCleared()

	if got := lookupGoSymbolSnapshot(goSymbolSnapshotCacheKey(dir, "clear-state")); got != nil {
		t.Fatalf("expected snapshot cache entry to be cleared, got %+v", got)
	}
}
