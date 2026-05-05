package review

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestReviewEvidenceBuilder_DisablesConfiguredFSMonitor(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	marker, helper := createReviewEvidenceMarkerScript(t, "fsmonitor", "")
	runGit(t, repo, "config", "core.fsmonitor", helper)
	writeTestFile(t, filepath.Join(repo, "keep.txt"), "fsmonitor safety\n")

	bundle, err := NewReviewEvidenceBuilder(repo, repo).BuildCurrentChanges(context.Background())
	if err != nil {
		t.Fatalf("BuildCurrentChanges() error = %v", err)
	}

	assertChangedFiles(t, bundle.ChangedFiles, []ReviewChangedFile{{
		Path:     "keep.txt",
		Status:   "M",
		Unstaged: true,
	}})
	assertReviewEvidenceMarkerAbsent(t, marker)
	assertReviewEvidenceMarkerCanBeInvoked(t, repo, marker, "status", "--short")
}

func TestReviewEvidenceBuilder_BlocksRepoControlledGitExecutableOnPATH(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	repoBin := filepath.Join(repo, "bin")
	marker := filepath.Join(t.TempDir(), "repo-git.marker")
	t.Setenv("REVIEW_EVIDENCE_MARKER", marker)
	createProbeTestScriptCommand(t, repoBin, "git", `printf invoked > "$REVIEW_EVIDENCE_MARKER"`)
	t.Setenv("PATH", strings.Join([]string{repoBin, os.Getenv("PATH")}, string(os.PathListSeparator)))

	_, err := NewReviewEvidenceBuilder(repo, repo).BuildCurrentChanges(context.Background())
	if err == nil {
		t.Fatal("BuildCurrentChanges() error = nil, want repo-controlled git to be blocked")
	}
	if !errors.Is(err, ErrHostReadOnlyBlocked) {
		t.Fatalf("BuildCurrentChanges() error = %v, want ErrHostReadOnlyBlocked", err)
	}
	assertFileAbsent(t, marker)
}

func TestReviewEvidenceBuilder_GitLookupIgnoresOutsideCWDRelativePATH(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	outsideCwd := t.TempDir()
	marker := filepath.Join(t.TempDir(), "outside-cwd-git.marker")
	t.Setenv("REVIEW_EVIDENCE_MARKER", marker)
	createProbeTestScriptCommand(t, outsideCwd, "git", `printf invoked > "$REVIEW_EVIDENCE_MARKER"`)
	t.Setenv("PATH", reviewEvidenceTestPathWithGit(t, "."))

	bundle, err := NewReviewEvidenceBuilder(repo, outsideCwd).BuildCurrentChanges(context.Background())
	if err != nil {
		t.Fatalf("BuildCurrentChanges() error = %v", err)
	}

	assertFileAbsent(t, marker)
	if bundle.CWD != filepath.Clean(outsideCwd) {
		t.Fatalf("CWD = %q, want outside cwd %q", bundle.CWD, filepath.Clean(outsideCwd))
	}
	wantRoot, err := filepath.EvalSymlinks(repo)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q) error = %v", repo, err)
	}
	if bundle.RepoRoot != filepath.Clean(wantRoot) {
		t.Fatalf("RepoRoot = %q, want %q", bundle.RepoRoot, filepath.Clean(wantRoot))
	}
}

func TestReviewEvidenceBuilder_RelativePATHEntriesResolveFromRepoRoot(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	outsideCwd := t.TempDir()
	repoBin := filepath.Join(repo, "bin")
	marker := filepath.Join(t.TempDir(), "repo-relative-git.marker")
	t.Setenv("REVIEW_EVIDENCE_MARKER", marker)
	createProbeTestScriptCommand(t, repoBin, "git", `printf invoked > "$REVIEW_EVIDENCE_MARKER"`)
	t.Setenv("PATH", reviewEvidenceTestPathWithGit(t, "bin"))

	_, err := NewReviewEvidenceBuilder(repo, outsideCwd).BuildCurrentChanges(context.Background())
	if err == nil {
		t.Fatal("BuildCurrentChanges() error = nil, want repo-relative git to be blocked")
	}
	if !errors.Is(err, ErrHostReadOnlyBlocked) {
		t.Fatalf("BuildCurrentChanges() error = %v, want ErrHostReadOnlyBlocked", err)
	}
	assertFileAbsent(t, marker)
}

func TestBuildReviewEvidenceGitEnvPinsOptionalLocks(t *testing.T) {
	got := buildReviewEvidenceGitEnv([]string{
		"PATH=/bin",
		"GIT_OPTIONAL_LOCKS=1",
		"git_optional_locks=true",
		"GIT_DIR=/tmp/repo/.git",
		"GIT_TRACE=/tmp/review.trace",
		"git_trace2_event=/tmp/review-trace.json",
		"GIT_TRACE_PACKET=1",
		"GIT_DIFF_OPTS=--unified=0",
	})

	assertStringSlice(t, got, []string{
		"PATH=/bin",
		"GIT_OPTIONAL_LOCKS=0",
	})
}

func TestReviewEvidenceGitProcessStreamsSeparateParsedOutputFromDiagnostics(t *testing.T) {
	streams := newReviewEvidenceGitProcessStreams(0)
	if _, err := streams.stdout.Write([]byte("M\x00keep.txt\x00")); err != nil {
		t.Fatalf("stdout Write() error = %v", err)
	}
	if _, err := streams.stderr.Write([]byte("warning: ignored diagnostic\n")); err != nil {
		t.Fatalf("stderr Write() error = %v", err)
	}

	got := streams.result()
	if got.parsedOutput != "M\x00keep.txt\x00" {
		t.Fatalf("parsedOutput = %q, want stdout only", got.parsedOutput)
	}
	if !strings.Contains(got.diagnostics, "warning: ignored diagnostic") || !strings.Contains(got.diagnostics, "M\x00keep.txt\x00") {
		t.Fatalf("diagnostics = %q, want stderr and stdout", got.diagnostics)
	}
}

func TestNewReviewEvidenceGitProcessUsesRepoRootWorkingDir(t *testing.T) {
	repoRoot := filepath.Clean(t.TempDir())
	safeBin := t.TempDir()
	createProbeTestExecutable(t, safeBin, "git")
	t.Setenv("PATH", safeBin)

	proc, err := newReviewEvidenceGitProcess(context.Background(), reviewEvidenceGitProcessRequest{
		repoRoot:       repoRoot,
		args:           []string{"status", "--short"},
		maxOutputBytes: 1024,
	})
	if err != nil {
		t.Fatalf("newReviewEvidenceGitProcess() error = %v", err)
	}
	if proc.Dir != repoRoot {
		t.Fatalf("proc.Dir = %q, want repo root %q", proc.Dir, repoRoot)
	}

	foundRepoRootArg := false
	for i := 0; i+1 < len(proc.Args); i++ {
		if proc.Args[i] == "-C" && proc.Args[i+1] == repoRoot {
			foundRepoRootArg = true
			break
		}
	}
	if !foundRepoRootArg {
		t.Fatalf("proc.Args = %#v, want -C %q", proc.Args, repoRoot)
	}
}

func TestBuildReviewEvidenceGitArgsAppliesConcreteRunnerPolicy(t *testing.T) {
	got := buildReviewEvidenceGitArgs("/repo", []string{"status", "--short"})

	assertStringSlice(t, got, []string{
		"-c", "core.fsmonitor=false",
		"-c", "core.untrackedCache=false",
		"-c", "diff.external=",
		"-c", "color.ui=false",
		"-c", "color.diff=false",
		"-c", "color.status=false",
		"-c", "diff.renames=true",
		"-C", "/repo",
		"status", "--short",
	})
}
