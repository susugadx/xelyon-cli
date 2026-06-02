package evidence

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestReviewEvidenceBuilder_CurrentChangesIgnoresAmbientPathspecEnvForRelatedCandidates(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())

	t.Setenv("GIT_LITERAL_PATHSPECS", "1")
	t.Setenv("GIT_GLOB_PATHSPECS", "1")
	t.Setenv("GIT_NOGLOB_PATHSPECS", "1")
	t.Setenv("GIT_ICASE_PATHSPECS", "1")

	writeTestFile(t, filepath.Join(repo, "probe", "probe.go"), `package probe

type Calculator struct{}

func Add(a, b int) int { return a + b }
`)

	bundle, err := NewReviewEvidenceBuilder(repo, repo).BuildCurrentChanges(context.Background())
	if err != nil {
		t.Fatalf("BuildCurrentChanges() error = %v", err)
	}

	requireReviewContextFile(t, bundle.RelatedContextFiles, "probe/probe_test.go")
	requireReviewSearchHit(t, bundle.RelatedSearchHits, "probe/probe_test.go")
}

func TestReviewEvidenceBuilder_CurrentChangesPropagatesRelatedCandidateListTruncation(t *testing.T) {
	repo := newProbeTestRepo(t, withProbeTestRepoNoLargeFile())
	writeTestFile(t, filepath.Join(repo, "probe", "probe.go"), `package probe

type Calculator struct{}

func Add(a, b int) int { return a + b }
`)

	fullCandidateList := "probe/probe_test.go\x00probe/partial_candidate.go\x00"
	limit := int64(len("probe/probe_test.go\x00probe/partial"))
	bundle, err := NewReviewEvidenceBuilder(repo, repo,
		WithReviewEvidenceLimits(ReviewEvidenceLimits{
			MaxCommandOutputBytes: limit,
		}),
		WithReviewEvidenceCommandRunner(fakeReviewEvidenceRunner{
			outputs: map[string]string{
				fakeReviewEvidenceGitKey(reviewDiffMetadataGitArgs(false, "--name-status", "-z")...): "M\x00probe/probe.go\x00",
				fakeReviewEvidenceGitKey(reviewRelatedCandidateListGitArgs()...):                     fullCandidateList,
			},
		}),
	).BuildCurrentChanges(context.Background())
	if err != nil {
		t.Fatalf("BuildCurrentChanges() error = %v", err)
	}

	if !bundle.RelatedCandidateListTruncated {
		t.Fatal("RelatedCandidateListTruncated = false, want true")
	}
	requireReviewContextFile(t, bundle.RelatedContextFiles, "probe/probe_test.go")
	if hasReviewContextFile(bundle.RelatedContextFiles, "probe/partial_candidate.go") {
		t.Fatalf("RelatedContextFiles = %#v, want partial candidate dropped", bundle.RelatedContextFiles)
	}

	input := BuildReviewEvidenceModelInput(bundle)
	if !input.TruncationFlags.RelatedCandidates {
		t.Fatalf("TruncationFlags.RelatedCandidates = false, want true")
	}
	markdown := RenderReviewEvidenceMarkdown(bundle)
	if !strings.Contains(markdown, `"related_candidates": true`) {
		t.Fatalf("markdown = %q, want related candidate truncation flag", markdown)
	}
}
