package config

import (
	"os/exec"
	"path/filepath"
	"testing"
)

func TestGitRootResolver_DoesNotCacheMiss(t *testing.T) {
	r := &gitRootResolver{}
	dir := t.TempDir()

	got := r.find(dir)
	if got != "" {
		t.Fatalf("expected empty git root in non-git dir, got %q", got)
	}

	cacheKey := normalizeGitRootCacheKey(dir)
	if _, ok := r.cache.Load(cacheKey); ok {
		t.Fatal("expected non-git miss to not be cached")
	}
}

func TestGitRootResolver_ReDetectsAfterGitInit(t *testing.T) {
	r := &gitRootResolver{}
	dir := t.TempDir()

	if got := r.find(dir); got != "" {
		t.Fatalf("expected empty git root in non-git dir, got %q", got)
	}

	cmd := exec.Command("git", "-C", dir, "init")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init failed: %v (%s)", err, string(out))
	}

	got := r.find(dir)
	want, err := filepath.Abs(dir)
	if err != nil {
		t.Fatalf("failed to resolve abs path: %v", err)
	}
	if filepath.Clean(got) != filepath.Clean(want) {
		t.Fatalf("expected git root %q, got %q", want, got)
	}

	cacheKey := normalizeGitRootCacheKey(dir)
	cached, ok := r.cache.Load(cacheKey)
	if !ok {
		t.Fatal("expected successful git root to be cached")
	}
	root, ok := cached.(string)
	if !ok {
		t.Fatalf("unexpected cache entry type: %#v", cached)
	}
	if filepath.Clean(root) != filepath.Clean(want) {
		t.Fatalf("expected cached root %q, got %q", want, root)
	}
}
