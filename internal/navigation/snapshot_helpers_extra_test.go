package navigation

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStoreGoSymbolSnapshot_ReplacesPreviousRootCacheKey(t *testing.T) {
	clearGoSymbolSnapshotCache()
	t.Cleanup(clearGoSymbolSnapshotCache)

	root := t.TempDir()
	oldKey := goSymbolSnapshotCacheKey(root, "old")
	newKey := goSymbolSnapshotCacheKey(root, "new")

	storeGoSymbolSnapshot(oldKey, &goSymbolSnapshot{
		RootPath: root,
		StateKey: "old",
		ByName:   map[string][]goSymbolSnapshotEntry{"Old": {{Name: "Old"}}},
	})
	storeGoSymbolSnapshot(newKey, &goSymbolSnapshot{
		RootPath: root,
		StateKey: "new",
		ByName:   map[string][]goSymbolSnapshotEntry{"New": {{Name: "New"}}},
	})

	if got := lookupGoSymbolSnapshot(oldKey); got != nil {
		t.Fatalf("lookupGoSymbolSnapshot(oldKey) = %+v, want nil after replacement", got)
	}
	if got := lookupGoSymbolSnapshot(newKey); got == nil || got.StateKey != "new" {
		t.Fatalf("lookupGoSymbolSnapshot(newKey) = %+v, want new snapshot", got)
	}
}

func TestClearGoSymbolSnapshotCacheWithKeys_ClearsStoredEntries(t *testing.T) {
	clearGoSymbolSnapshotCache()
	t.Cleanup(clearGoSymbolSnapshotCache)

	rootA := t.TempDir()
	rootB := t.TempDir()
	keyA := goSymbolSnapshotCacheKey(rootA, "state-a")
	keyB := goSymbolSnapshotCacheKey(rootB, "state-b")

	storeGoSymbolSnapshot(keyA, &goSymbolSnapshot{RootPath: rootA, StateKey: "state-a"})
	storeGoSymbolSnapshot(keyB, &goSymbolSnapshot{RootPath: rootB, StateKey: "state-b"})

	clearGoSymbolSnapshotCacheWithKeys([]string{"pkg/a.go"})

	if got := lookupGoSymbolSnapshot(keyA); got != nil {
		t.Fatalf("snapshot A still exists: %+v", got)
	}
	if got := lookupGoSymbolSnapshot(keyB); got != nil {
		t.Fatalf("snapshot B still exists: %+v", got)
	}
	if _, ok := goSymbolSnapshotRootKeys.Load(filepath.Clean(rootA)); ok {
		t.Fatalf("root key for %q should be cleared", rootA)
	}
}

func TestBuildSnapshotPathMatcher_FileAndDirectoryHints(t *testing.T) {
	root := t.TempDir()
	invocationDir := filepath.Join(root, "pkg")
	filePath := filepath.Join(invocationDir, "run.go")
	subFilePath := filepath.Join(invocationDir, "sub", "other.go")
	if err := os.MkdirAll(filepath.Dir(subFilePath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filePath, []byte("package pkg\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(subFilePath, []byte("package sub\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if matcher := buildSnapshotPathMatcher(root, invocationDir, root); matcher != nil {
		t.Fatal("buildSnapshotPathMatcher(root hint) should return nil")
	}

	fileMatcher := buildSnapshotPathMatcher(root, invocationDir, "run.go")
	if fileMatcher == nil || !fileMatcher("pkg/run.go") || fileMatcher("pkg/sub/other.go") {
		t.Fatal("file matcher should match only the resolved file")
	}

	dirMatcher := buildSnapshotPathMatcher(root, "", invocationDir)
	if dirMatcher == nil {
		t.Fatal("directory matcher should not be nil for nested directory hint")
	}
	if !dirMatcher("pkg/run.go") || !dirMatcher("pkg/sub/other.go") || dirMatcher("other.go") {
		t.Fatal("directory matcher should match only files under hinted directory")
	}
}

func TestAbsoluteToSnapshotRel_RejectsOutsideRoot(t *testing.T) {
	root := t.TempDir()
	inside := filepath.Join(root, "pkg", "run.go")
	outside := filepath.Join(filepath.Dir(root), "outside.go")

	if rel, ok := absoluteToSnapshotRel(root, inside); !ok || rel != "pkg/run.go" {
		t.Fatalf("absoluteToSnapshotRel(inside) = (%q, %v), want (%q, true)", rel, ok, "pkg/run.go")
	}
	if rel, ok := absoluteToSnapshotRel(root, outside); ok || rel != "" {
		t.Fatalf("absoluteToSnapshotRel(outside) = (%q, %v), want empty false", rel, ok)
	}
}

func TestSortSymbolCandidates_OrdersByPathLineAndDisplayName(t *testing.T) {
	candidates := []SymbolCandidate{
		{File: "b.go", Line: 2, EndLine: 2, Name: "Build"},
		{File: "a.go", Line: 3, EndLine: 4, Name: "Build", Kind: "method", Receiver: "Config"},
		{File: "a.go", Line: 3, EndLine: 4, Name: "Build", Kind: "method", Receiver: "*Config"},
		{File: "a.go", Line: 1, EndLine: 1, Name: "Alpha"},
	}

	sortSymbolCandidates(candidates)

	got := []string{
		candidates[0].File + ":" + candidateDisplayName(candidates[0]),
		candidates[1].File + ":" + candidateDisplayName(candidates[1]),
		candidates[2].File + ":" + candidateDisplayName(candidates[2]),
		candidates[3].File + ":" + candidateDisplayName(candidates[3]),
	}
	want := []string{
		"a.go:Alpha",
		"a.go:(*Config).Build",
		"a.go:Config.Build",
		"b.go:Build",
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("sorted[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
