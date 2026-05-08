package review

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestReviewEvidenceBuilder_CommandFailureReturnsError(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	wantErr := errors.New("status failed")

	_, err := NewReviewEvidenceBuilder(repo, repo, WithReviewEvidenceCommandRunner(fakeReviewEvidenceRunner{
		failures: map[string]error{
			fakeReviewEvidenceGitKey(reviewStatusShortGitArgs()...): wantErr,
		},
	})).BuildCurrentChanges(context.Background())
	if !errors.Is(err, wantErr) {
		t.Fatalf("BuildCurrentChanges() error = %v, want %v", err, wantErr)
	}
}

func TestReviewEvidenceBuilder_CommandOutputLimitTruncatesDiff(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	writeTestFile(t, filepath.Join(repo, "keep.txt"), strings.Repeat("x", 128)+"\n")

	bundle, err := NewReviewEvidenceBuilder(repo, repo, WithReviewEvidenceLimits(ReviewEvidenceLimits{
		MaxCommandOutputBytes: 32,
	})).BuildCurrentChanges(context.Background())
	if err != nil {
		t.Fatalf("BuildCurrentChanges() error = %v", err)
	}

	unstaged := diffEvidenceBySource(t, bundle, reviewDiffEvidenceSourceUnstaged)
	if !unstaged.DiffTruncated {
		t.Fatalf("DiffTruncated = false, want true; diff=%q", unstaged.Diff)
	}
	if len(unstaged.Diff) != 32 {
		t.Fatalf("len(Diff) = %d, want 32", len(unstaged.Diff))
	}
}
