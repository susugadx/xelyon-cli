package evidence

import (
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func newReviewEvidenceRenderTestBundle(t *testing.T) ReviewEvidenceBundle {
	t.Helper()

	repo := filepath.Clean(t.TempDir())
	cwd := filepath.Join(repo, "src", "work")

	return ReviewEvidenceBundle{
		TargetKind:           TargetCurrentChanges,
		RepoRoot:             repo,
		CWD:                  cwd,
		StatusShort:          " M src/main.go\n?? notes.md\n",
		StatusShortTruncated: true,
		Diffs: []ReviewDiffEvidence{
			{
				Source:              reviewDiffEvidenceSourceUnstaged,
				Stat:                " src/main.go | 2 +-\n",
				StatTruncated:       true,
				NameStatus:          "M\tsrc/main.go\n",
				NameStatusTruncated: true,
				Diff:                "diff --git a/src/main.go b/src/main.go\n+changed\n",
				DiffTruncated:       true,
			},
			{
				Source:     reviewDiffEvidenceSourceStaged,
				Stat:       " README.md | 1 +\n",
				NameStatus: "M\tREADME.md\n",
				Diff:       "diff --git a/README.md b/README.md\n+docs\n",
			},
		},
		ChangedFiles: []ReviewChangedFile{
			{
				Path:     filepath.Join(repo, "src", "main.go"),
				OldPath:  filepath.Join(repo, "src", "old.go"),
				Status:   "R100",
				Staged:   true,
				Unstaged: true,
			},
		},
		ChangedFileContext: []ReviewContextFileEvidence{
			{
				Path:      filepath.Join(repo, "src", "main.go"),
				Role:      reviewContextFileRoleChanged,
				Content:   "package src\n\nfunc Run() {}\n",
				Truncated: true,
				SizeBytes: 256,
				ReadBytes: 128,
			},
		},
		RelatedContextFiles: []ReviewContextFileEvidence{
			{
				Path:      filepath.Join(repo, "src", "main_test.go"),
				Role:      reviewContextFileRoleRelatedTest,
				Content:   "package src\n\nfunc TestRun(t *testing.T) {}\n",
				SizeBytes: 96,
				ReadBytes: 96,
			},
		},
		RelatedSearchHits: []ReviewRelatedSearchHit{
			{
				Path:    filepath.Join(repo, "src", "helper.go"),
				Line:    7,
				Snippet: "func helper() { Run() }",
				Reason:  "symbol:Run",
			},
		},
		UntrackedFiles: []ReviewUntrackedFile{
			{
				Path:      filepath.Join(repo, "notes.md"),
				Snapshot:  "draft notes\n",
				Truncated: true,
				SizeBytes: 64,
				ReadBytes: 32,
			},
		},
		RelatedCandidateListTruncated: true,
		RelatedSearchTruncated:        true,
		UntrackedListTruncated:        true,
		UntrackedSnapshotsTruncated:   true,
		RuleFiles: []ReviewRuleFileEvidence{
			{
				Path:      filepath.Join(repo, "AGENTS.md"),
				Content:   "rule text\n",
				Truncated: true,
				SizeBytes: 128,
			},
		},
		Inventory: ReviewChangeInventory{
			Production: []string{filepath.Join(repo, "src", "main.go")},
			Docs:       []string{"README.md"},
			RenamedFiles: []string{
				filepath.Join(repo, "src", "main.go"),
			},
			Untracked: []string{filepath.Join(repo, "notes.md")},
		},
		Limits: ReviewEvidenceLimits{
			MaxCommandOutputBytes:      1024,
			MaxUntrackedFileBytes:      64,
			MaxRuleFileBytes:           128,
			MaxTotalUntrackedBytes:     256,
			MaxUntrackedFiles:          3,
			MaxContextFileBytes:        512,
			MaxTotalContextBytes:       1024,
			MaxContextFiles:            4,
			MaxRelatedSearchTerms:      5,
			MaxRelatedSearchFiles:      6,
			MaxTotalRelatedSearchBytes: 2048,
			MaxRelatedSearchFileBytes:  256,
			MaxRelatedSearchHits:       6,
			MaxSearchSnippetBytes:      120,
			CommandTimeout:             1500 * time.Millisecond,
		},
	}
}

func assertReviewEvidenceSubstringsInOrder(t *testing.T, text string, wants []string) {
	t.Helper()

	offset := 0
	for _, want := range wants {
		index := strings.Index(text[offset:], want)
		if index < 0 {
			t.Fatalf("text after offset %d does not contain %q:\n%s", offset, want, text)
		}
		offset += index + len(want)
	}
}
