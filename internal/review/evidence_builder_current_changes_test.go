package review

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestReviewEvidenceBuilder_CurrentChangesNoChanges(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())

	bundle, err := NewReviewEvidenceBuilder(repo, repo).BuildCurrentChanges(context.Background())
	if err != nil {
		t.Fatalf("BuildCurrentChanges() error = %v", err)
	}

	if bundle.TargetKind != TargetCurrentChanges {
		t.Fatalf("TargetKind = %q, want %q", bundle.TargetKind, TargetCurrentChanges)
	}
	wantRepoRoot, err := filepath.EvalSymlinks(repo)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q) error = %v", repo, err)
	}
	if bundle.RepoRoot != filepath.Clean(wantRepoRoot) || bundle.CWD != filepath.Clean(repo) {
		t.Fatalf("repo dirs = (%q, %q), want (%q, %q)", bundle.RepoRoot, bundle.CWD, filepath.Clean(wantRepoRoot), filepath.Clean(repo))
	}
	if bundle.StatusShort != "" {
		t.Fatalf("StatusShort = %q, want empty", bundle.StatusShort)
	}
	if len(bundle.ChangedFiles) != 0 {
		t.Fatalf("ChangedFiles = %#v, want empty", bundle.ChangedFiles)
	}
	if len(bundle.UntrackedFiles) != 0 {
		t.Fatalf("UntrackedFiles = %#v, want empty", bundle.UntrackedFiles)
	}
	if bundle.UntrackedListTruncated {
		t.Fatal("UntrackedListTruncated = true, want false")
	}
	if bundle.UntrackedSnapshotsTruncated {
		t.Fatal("UntrackedSnapshotsTruncated = true, want false")
	}
	if got, want := len(bundle.Diffs), 2; got != want {
		t.Fatalf("len(Diffs) = %d, want %d", got, want)
	}
}

func TestReviewEvidenceBuilder_CurrentChangesMarksUnstagedOnlyChangedFile(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	writeTestFile(t, filepath.Join(repo, "keep.txt"), "unstaged\n")

	bundle, err := NewReviewEvidenceBuilder(repo, repo).BuildCurrentChanges(context.Background())
	if err != nil {
		t.Fatalf("BuildCurrentChanges() error = %v", err)
	}

	assertChangedFiles(t, bundle.ChangedFiles, []ReviewChangedFile{{
		Path:     "keep.txt",
		Status:   "M",
		Unstaged: true,
	}})
}

func TestReviewEvidenceBuilder_CurrentChangesMarksStagedOnlyChangedFile(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	writeTestFile(t, filepath.Join(repo, "keep.txt"), "staged\n")
	runGit(t, repo, "add", "keep.txt")

	bundle, err := NewReviewEvidenceBuilder(repo, repo).BuildCurrentChanges(context.Background())
	if err != nil {
		t.Fatalf("BuildCurrentChanges() error = %v", err)
	}

	assertChangedFiles(t, bundle.ChangedFiles, []ReviewChangedFile{{
		Path:   "keep.txt",
		Status: "M",
		Staged: true,
	}})
}

func TestReviewEvidenceBuilder_CurrentChangesSeparatesStagedAndUnstaged(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	path := filepath.Join(repo, "keep.txt")

	writeTestFile(t, path, "staged\n")
	runGit(t, repo, "add", "keep.txt")
	writeTestFile(t, path, "unstaged\n")

	bundle, err := NewReviewEvidenceBuilder(repo, repo).BuildCurrentChanges(context.Background())
	if err != nil {
		t.Fatalf("BuildCurrentChanges() error = %v", err)
	}

	assertChangedFiles(t, bundle.ChangedFiles, []ReviewChangedFile{{
		Path:     "keep.txt",
		Status:   "M",
		Staged:   true,
		Unstaged: true,
	}})
	unstaged := diffEvidenceBySource(t, bundle, reviewDiffEvidenceSourceUnstaged)
	staged := diffEvidenceBySource(t, bundle, reviewDiffEvidenceSourceStaged)
	if !strings.Contains(unstaged.Diff, "+unstaged") {
		t.Fatalf("unstaged diff missing worktree content: %q", unstaged.Diff)
	}
	if !strings.Contains(staged.Diff, "+staged") {
		t.Fatalf("staged diff missing index content: %q", staged.Diff)
	}
	if !containsString(bundle.Inventory.Production, "keep.txt") {
		t.Fatalf("Inventory.Production = %#v, want keep.txt", bundle.Inventory.Production)
	}
}

func TestReviewEvidenceBuilder_CurrentChangesCanonicalizesSymlinkRepoRoot(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	writeTestFile(t, filepath.Join(repo, "notes.txt"), "review notes\n")

	linkRoot := filepath.Join(t.TempDir(), "repo-link")
	createReviewEvidenceSymlink(t, repo, linkRoot)

	bundle, err := NewReviewEvidenceBuilder(linkRoot, linkRoot).BuildCurrentChanges(context.Background())
	if err != nil {
		t.Fatalf("BuildCurrentChanges() error = %v", err)
	}

	wantRoot, err := filepath.EvalSymlinks(repo)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q) error = %v", repo, err)
	}
	if bundle.RepoRoot != filepath.Clean(wantRoot) {
		t.Fatalf("RepoRoot = %q, want canonical %q", bundle.RepoRoot, filepath.Clean(wantRoot))
	}
	if bundle.CWD != filepath.Clean(linkRoot) {
		t.Fatalf("CWD = %q, want lexical cwd %q", bundle.CWD, filepath.Clean(linkRoot))
	}

	got := requireReviewUntrackedFile(t, bundle, "notes.txt")
	if got.Snapshot != "review notes\n" {
		t.Fatalf("UntrackedFiles[notes.txt] = %#v, want snapshot from symlink repo root", got)
	}
}

func TestReviewEvidenceBuilder_CurrentChangesReportsGitRename(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	runGit(t, repo, "mv", "keep.txt", "renamed.txt")

	bundle, err := NewReviewEvidenceBuilder(repo, repo).BuildCurrentChanges(context.Background())
	if err != nil {
		t.Fatalf("BuildCurrentChanges() error = %v", err)
	}

	assertChangedFiles(t, bundle.ChangedFiles, []ReviewChangedFile{{
		Path:    "renamed.txt",
		OldPath: "keep.txt",
		Status:  "R100",
		Staged:  true,
	}})
	assertStringSlice(t, bundle.Inventory.RenamedFiles, []string{"renamed.txt"})
}

func TestReviewEvidenceBuilder_CurrentChangesReportsStagedAddDeleteInventory(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	writeTestFile(t, filepath.Join(repo, "new.go"), "package main\n")
	runGit(t, repo, "add", "new.go")
	runGit(t, repo, "rm", "keep.txt")

	bundle, err := NewReviewEvidenceBuilder(repo, repo).BuildCurrentChanges(context.Background())
	if err != nil {
		t.Fatalf("BuildCurrentChanges() error = %v", err)
	}

	assertStringSlice(t, bundle.Inventory.NewFiles, []string{"new.go"})
	assertStringSlice(t, bundle.Inventory.DeletedFiles, []string{"keep.txt"})
}
