package review

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestReviewEvidenceBuilder_CurrentChangesIgnoresParentGitIndexFile(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	alternateIndex := createReviewEvidenceIndexAtHead(t, repo)

	writeTestFile(t, filepath.Join(repo, "keep.txt"), "staged through real index\n")
	runGit(t, repo, "add", "keep.txt")
	t.Setenv("GIT_INDEX_FILE", alternateIndex)

	bundle, err := NewReviewEvidenceBuilder(repo, repo).BuildCurrentChanges(context.Background())
	if err != nil {
		t.Fatalf("BuildCurrentChanges() error = %v", err)
	}

	assertChangedFiles(t, bundle.ChangedFiles, []ReviewChangedFile{{
		Path:   "keep.txt",
		Status: "M",
		Staged: true,
	}})
	staged := diffEvidenceBySource(t, bundle, reviewDiffEvidenceSourceStaged)
	if !strings.Contains(staged.Diff, "+staged through real index") {
		t.Fatalf("staged diff = %q, want real index content", staged.Diff)
	}
}

func TestReviewEvidenceBuilder_CurrentChangesIgnoresOutputAffectingGitConfig(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	runGit(t, repo, "config", "color.ui", "always")
	runGit(t, repo, "config", "color.diff", "always")
	runGit(t, repo, "config", "color.status", "always")
	runGit(t, repo, "config", "diff.renames", "false")
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

	assertReviewEvidenceNoANSI(t, "StatusShort", bundle.StatusShort)
	staged := diffEvidenceBySource(t, bundle, reviewDiffEvidenceSourceStaged)
	if got, want := staged.NameStatus, "R100\tkeep.txt\trenamed.txt\n"; got != want {
		t.Fatalf("staged NameStatus = %q, want %q", got, want)
	}
	assertReviewEvidenceNoANSI(t, "staged Stat", staged.Stat)
	assertReviewEvidenceNoANSI(t, "staged NameStatus", staged.NameStatus)
	assertReviewEvidenceNoANSI(t, "staged Diff", staged.Diff)
}

func TestReviewEvidenceBuilder_CurrentChangesPinsStatusSemantics(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	runGit(t, repo, "config", "status.showUntrackedFiles", "no")
	runGit(t, repo, "config", "status.renames", "false")
	runGit(t, repo, "mv", "keep.txt", "renamed.txt")
	writeTestFile(t, filepath.Join(repo, "notes.txt"), "untracked\n")

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
	if !strings.Contains(bundle.StatusShort, "R  keep.txt -> renamed.txt\n") {
		t.Fatalf("StatusShort = %q, want staged rename", bundle.StatusShort)
	}
	if !strings.Contains(bundle.StatusShort, "?? notes.txt\n") {
		t.Fatalf("StatusShort = %q, want untracked notes.txt", bundle.StatusShort)
	}
}

func TestReviewEvidenceBuilder_CurrentChangesPinsDiffContext(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	writeTestFile(t, filepath.Join(repo, "keep.txt"), "line1\nline2\nline3\nline4\nline5\n")
	runGit(t, repo, "add", "keep.txt")
	runGit(t, repo, "commit", "-m", "add context fixture")
	runGit(t, repo, "config", "diff.context", "0")
	t.Setenv("GIT_DIFF_OPTS", "--unified=0")
	writeTestFile(t, filepath.Join(repo, "keep.txt"), "line1\nline2\nchanged\nline4\nline5\n")

	bundle, err := NewReviewEvidenceBuilder(repo, repo).BuildCurrentChanges(context.Background())
	if err != nil {
		t.Fatalf("BuildCurrentChanges() error = %v", err)
	}

	unstaged := diffEvidenceBySource(t, bundle, reviewDiffEvidenceSourceUnstaged)
	if !strings.Contains(unstaged.Diff, "\n line2\n") || !strings.Contains(unstaged.Diff, "\n line4\n") {
		t.Fatalf("unstaged Diff = %q, want surrounding context despite diff config/env", unstaged.Diff)
	}
}

func TestReviewEvidenceBuilder_CurrentChangesScrubsGitTraceEnv(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	writeTestFile(t, filepath.Join(repo, "keep.txt"), "trace env safety\n")
	tracePath := filepath.Join(t.TempDir(), "review.trace")
	t.Setenv("GIT_TRACE", tracePath)

	bundle, err := NewReviewEvidenceBuilder(repo, repo).BuildCurrentChanges(context.Background())
	if err != nil {
		t.Fatalf("BuildCurrentChanges() error = %v", err)
	}

	assertChangedFiles(t, bundle.ChangedFiles, []ReviewChangedFile{{
		Path:     "keep.txt",
		Status:   "M",
		Unstaged: true,
	}})
	assertFileAbsent(t, tracePath)
}

func TestReviewEvidenceBuilder_CurrentChangesKeepsGitDiagnosticsOutOfNameStatus(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unreadable XDG_CONFIG_HOME fixture is unix-only")
	}

	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	writeTestFile(t, filepath.Join(repo, "keep.txt"), "stderr safety\n")
	xdgConfigHome := filepath.Join(t.TempDir(), "xdg-config-home")
	if err := os.MkdirAll(xdgConfigHome, 0o755); err != nil {
		t.Fatalf("MkdirAll(%q) error = %v", xdgConfigHome, err)
	}
	if err := os.Chmod(xdgConfigHome, 0); err != nil {
		t.Fatalf("Chmod(%q) error = %v", xdgConfigHome, err)
	}
	t.Cleanup(func() {
		_ = os.Chmod(xdgConfigHome, 0o755)
	})
	t.Setenv("XDG_CONFIG_HOME", xdgConfigHome)

	bundle, err := NewReviewEvidenceBuilder(repo, repo).BuildCurrentChanges(context.Background())
	if err != nil {
		t.Fatalf("BuildCurrentChanges() error = %v", err)
	}

	assertChangedFiles(t, bundle.ChangedFiles, []ReviewChangedFile{{
		Path:     "keep.txt",
		Status:   "M",
		Unstaged: true,
	}})
	unstaged := diffEvidenceBySource(t, bundle, reviewDiffEvidenceSourceUnstaged)
	if got, want := unstaged.NameStatus, "M\tkeep.txt\n"; got != want {
		t.Fatalf("unstaged NameStatus = %q, want %q", got, want)
	}
	if strings.Contains(unstaged.NameStatus, "warning:") {
		t.Fatalf("unstaged NameStatus contains git diagnostic: %q", unstaged.NameStatus)
	}
}
