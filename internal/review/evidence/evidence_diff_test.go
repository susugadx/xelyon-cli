package evidence

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestReviewEvidenceBuilder_DiffDisablesConfiguredExternalDiff(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	marker, helper := createReviewEvidenceMarkerScript(t, "external-diff", "")
	runGit(t, repo, "config", "diff.external", helper)
	writeTestFile(t, filepath.Join(repo, "keep.txt"), "external diff safety\n")

	bundle, err := NewReviewEvidenceBuilder(repo, repo).BuildCurrentChanges(context.Background())
	if err != nil {
		t.Fatalf("BuildCurrentChanges() error = %v", err)
	}

	assertReviewEvidenceDiffContainsKeepChange(t, bundle, "+external diff safety")
	assertReviewEvidenceMarkerAbsent(t, marker)
	assertReviewEvidenceMarkerCanBeInvoked(t, repo, marker, "diff")
}

func TestReviewEvidenceBuilder_DiffDisablesExternalDiffEnv(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	marker, helper := createReviewEvidenceMarkerScript(t, "external-diff-env", "")
	t.Setenv("GIT_EXTERNAL_DIFF", helper)
	writeTestFile(t, filepath.Join(repo, "keep.txt"), "external diff env safety\n")

	bundle, err := NewReviewEvidenceBuilder(repo, repo).BuildCurrentChanges(context.Background())
	if err != nil {
		t.Fatalf("BuildCurrentChanges() error = %v", err)
	}

	assertReviewEvidenceDiffContainsKeepChange(t, bundle, "+external diff env safety")
	assertReviewEvidenceMarkerAbsent(t, marker)
	assertReviewEvidenceMarkerCanBeInvoked(t, repo, marker, "diff")
}

func TestReviewEvidenceBuilder_DiffDisablesTextconv(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	marker, helper := createReviewEvidenceMarkerScript(t, "textconv", `cat "$1"`)
	writeTestFile(t, filepath.Join(repo, ".gitattributes"), "keep.txt diff=marker\n")
	runGit(t, repo, "add", ".gitattributes")
	runGit(t, repo, "commit", "-m", "add attributes")
	runGit(t, repo, "config", "diff.marker.textconv", helper)
	writeTestFile(t, filepath.Join(repo, "keep.txt"), "textconv safety\n")

	bundle, err := NewReviewEvidenceBuilder(repo, repo).BuildCurrentChanges(context.Background())
	if err != nil {
		t.Fatalf("BuildCurrentChanges() error = %v", err)
	}

	assertReviewEvidenceDiffContainsKeepChange(t, bundle, "+textconv safety")
	assertReviewEvidenceMarkerAbsent(t, marker)
	assertReviewEvidenceMarkerCanBeInvoked(t, repo, marker, "diff", "--textconv", "--", "keep.txt")
}

func TestReviewEvidenceDiffParsingDropsTruncatedPartialRecord(t *testing.T) {
	entries := parseReviewNameStatusEntries("M\x00keep.txt\x00A\x00partial", true)
	got := buildReviewChangedFiles(entries, nil)

	assertChangedFiles(t, got, []ReviewChangedFile{{
		Path:     "keep.txt",
		Status:   "M",
		Unstaged: true,
	}})
}

func TestReviewEvidenceDiffParsingDropsTruncatedIncompleteRenameRecord(t *testing.T) {
	entries := parseReviewNameStatusEntries("M\x00keep.txt\x00R100\x00old.txt\x00new", true)
	got := buildReviewChangedFiles(entries, nil)

	assertChangedFiles(t, got, []ReviewChangedFile{{
		Path:     "keep.txt",
		Status:   "M",
		Unstaged: true,
	}})
}

func assertReviewEvidenceDiffContainsKeepChange(t *testing.T, bundle ReviewEvidenceBundle, wantAddedLine string) {
	t.Helper()

	unstaged := diffEvidenceBySource(t, bundle, reviewDiffEvidenceSourceUnstaged)
	if !strings.Contains(unstaged.Stat, "keep.txt") {
		t.Fatalf("unstaged Stat = %q, want keep.txt", unstaged.Stat)
	}
	if !strings.Contains(unstaged.NameStatus, "M\tkeep.txt\n") {
		t.Fatalf("unstaged NameStatus = %q, want modified keep.txt", unstaged.NameStatus)
	}
	if !strings.Contains(unstaged.Diff, wantAddedLine) {
		t.Fatalf("unstaged Diff = %q, want %q", unstaged.Diff, wantAddedLine)
	}
}
