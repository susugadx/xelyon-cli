package review

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	reviewprobe "github.com/susugadx/xelyon-cli/internal/review/probe"
)

func TestResolveReviewRepoRootResolvesRepoRoot(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())

	got, err := ResolveReviewRepoRoot(context.Background(), repo)
	if err != nil {
		t.Fatalf("ResolveReviewRepoRoot() error = %v", err)
	}
	if got != filepath.Clean(repo) {
		t.Fatalf("ResolveReviewRepoRoot() = %q, want %q", got, filepath.Clean(repo))
	}
}

func TestResolveReviewRepoRootResolvesFromSubdir(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	subdir := filepath.Join(repo, "probe")

	got, err := ResolveReviewRepoRoot(context.Background(), subdir)
	if err != nil {
		t.Fatalf("ResolveReviewRepoRoot() error = %v", err)
	}
	if got != filepath.Clean(repo) {
		t.Fatalf("ResolveReviewRepoRoot() = %q, want %q", got, filepath.Clean(repo))
	}
}

func TestResolveReviewRepoRootBlocksCurrentDirFakeGitOnRelativePATH(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	marker := filepath.Join(t.TempDir(), "repo-root-fake-git.marker")
	t.Setenv("REVIEW_REPO_ROOT_MARKER", marker)
	createProbeTestScriptCommand(t, repo, "git", `printf invoked > "$REVIEW_REPO_ROOT_MARKER"`)
	t.Setenv("PATH", reviewEvidenceTestPathWithGit(t, "."))

	_, err := ResolveReviewRepoRoot(context.Background(), repo)
	if err == nil {
		t.Fatal("ResolveReviewRepoRoot() error = nil, want repo-controlled git to be blocked")
	}
	if !errors.Is(err, reviewprobe.ErrHostReadOnlyBlocked) {
		t.Fatalf("ResolveReviewRepoRoot() error = %v, want ErrHostReadOnlyBlocked", err)
	}
	assertFileAbsent(t, marker)
}

func TestResolveReviewRepoRootBlocksRepoRelativeFakeGitFromSubdir(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	subdir := filepath.Join(repo, "probe")
	repoBin := filepath.Join(repo, "bin")
	marker := filepath.Join(t.TempDir(), "repo-root-bin-fake-git.marker")
	t.Setenv("REVIEW_REPO_ROOT_MARKER", marker)
	createProbeTestScriptCommand(t, repoBin, "git", `printf invoked > "$REVIEW_REPO_ROOT_MARKER"`)
	t.Setenv("PATH", reviewEvidenceTestPathWithGit(t, "bin"))

	_, err := ResolveReviewRepoRoot(context.Background(), subdir)
	if err == nil {
		t.Fatal("ResolveReviewRepoRoot() error = nil, want repo-controlled git to be blocked")
	}
	if !errors.Is(err, reviewprobe.ErrHostReadOnlyBlocked) {
		t.Fatalf("ResolveReviewRepoRoot() error = %v, want ErrHostReadOnlyBlocked", err)
	}
	assertFileAbsent(t, marker)
}

func TestResolveReviewRepoRootReturnsHelpfulErrorOutsideGitRepo(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("PATH", reviewEvidenceTestPathWithGit(t))

	_, err := ResolveReviewRepoRoot(context.Background(), dir)
	if err == nil {
		t.Fatal("ResolveReviewRepoRoot() error = nil, want non-repository error")
	}
	if got := err.Error(); !strings.Contains(got, "review run resolve repo root") {
		t.Fatalf("ResolveReviewRepoRoot() error = %q, want review run resolve repo root", got)
	}
}
