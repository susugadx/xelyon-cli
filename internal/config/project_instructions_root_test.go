package config

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestLoadProjectInstructionBundle_NoMarkersUsesCwdFallbackSource(t *testing.T) {
	dir := t.TempDir()

	bundle := loadProjectInstructionBundleForDirOrFatal(t, DefaultConfig(), dir)
	if bundle.RootPath != dir {
		t.Fatalf("RootPath = %q, want cwd %q", bundle.RootPath, dir)
	}
	if bundle.RootSource != ProjectInstructionRootSourceFallbackCWD {
		t.Fatalf("RootSource = %q, want %q", bundle.RootSource, ProjectInstructionRootSourceFallbackCWD)
	}
	if bundle.HasProjectRoot() {
		t.Fatal("HasProjectRoot() = true, want false for cwd fallback")
	}
}

func TestLoadProjectInstructionBundle_GuidanceRootSource(t *testing.T) {
	root := t.TempDir()
	cwd := filepath.Join(root, "nested")
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "AGENTS.md"), "# guidance\n")

	bundle := loadProjectInstructionBundleForDirOrFatal(t, DefaultConfig(), cwd)
	if bundle.RootPath != root {
		t.Fatalf("RootPath = %q, want guidance root %q", bundle.RootPath, root)
	}
	if bundle.RootSource != ProjectInstructionRootSourceGuidance {
		t.Fatalf("RootSource = %q, want %q", bundle.RootSource, ProjectInstructionRootSourceGuidance)
	}
	if got, ok := bundle.ProjectRootPath(); !ok || got != root {
		t.Fatalf("ProjectRootPath() = %q, %v; want %q, true", got, ok, root)
	}
}

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
