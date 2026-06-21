package modelinput

import (
	"path/filepath"
	"strings"
	"testing"

	reviewevidence "github.com/susugadx/xelyon-cli/internal/review/evidence"
)

func TestRenderReviewEvidenceMarkdownSectionOrderAndPathRedaction(t *testing.T) {
	bundle := newReviewEvidenceRenderTestBundle(t)

	markdown := RenderReviewEvidenceMarkdown(bundle)

	assertReviewEvidenceSubstringsInOrder(t, markdown, []string{
		"## target kind\n",
		"## repo root display\n",
		"## cwd display\n",
		"## git status --short\n",
		"## change inventory\n",
		"## review pressure signals\n",
		"## generic impact candidates\n",
		"## changed files\n",
		"## changed file context\n",
		"## related tests/context files\n",
		"## related search hits\n",
		"## rule files\n",
		"## diffs\n",
		"## untracked files\n",
		"## limits\n",
		"## truncation flags\n",
	})
	assertReviewEvidenceSubstringsInOrder(t, markdown, []string{
		"### diff: unstaged\n",
		"### diff: staged\n",
	})

	if !strings.Contains(markdown, "<repo_root>") {
		t.Fatalf("markdown = %q, want repo root placeholder", markdown)
	}
	if !strings.Contains(markdown, "src/work") {
		t.Fatalf("markdown = %q, want repo-relative cwd display", markdown)
	}
	if strings.Contains(markdown, bundle.RepoRoot) || strings.Contains(markdown, bundle.CWD) {
		t.Fatalf("markdown leaked absolute path: %q", markdown)
	}
	for _, want := range []string{
		`"status_short": true`,
		`"untracked_list": true`,
		`"related_candidates": true`,
		`"related_search": true`,
		`"untracked_snapshots": true`,
		`"diff": true`,
		`"path": "AGENTS.md"`,
		`"truncated": true`,
	} {
		if !strings.Contains(markdown, want) {
			t.Fatalf("markdown = %q, want truncation flag fragment %q", markdown, want)
		}
	}
}

func TestRenderReviewEvidenceMarkdownIncludesGenericImpactCandidates(t *testing.T) {
	bundle := newReviewEvidenceRenderTestBundle(t)
	repo := bundle.RepoRoot
	bundle.GenericImpactCandidates = reviewevidence.ReviewGenericImpactCandidates{
		Tokens: []string{"/review"},
		Candidates: []reviewevidence.ReviewGenericImpactCandidate{
			{
				Path:    filepath.Join(repo, "docs/commands.md"),
				Role:    reviewevidence.ReviewGenericImpactRoleDocsReference,
				Reason:  "docs mention " + filepath.Join(repo, "cmd", "review.ts"),
				Token:   "/review",
				Line:    3,
				Snippet: "Run " + filepath.Join(repo, "cmd", "review.ts") + " with /review.",
			},
		},
	}

	markdown := RenderReviewEvidenceMarkdown(bundle)

	if !strings.Contains(markdown, "## generic impact candidates\n") {
		t.Fatalf("markdown = %q, want generic impact candidates section", markdown)
	}
	for _, want := range []string{
		`"path": "docs/commands.md"`,
		`"role": "docs_reference"`,
		`"token": "/review"`,
		`"line": 3`,
		`<repo_root>/cmd/review.ts`,
	} {
		if !strings.Contains(markdown, want) {
			t.Fatalf("markdown = %q, want %q", markdown, want)
		}
	}
	if strings.Contains(markdown, repo) {
		t.Fatalf("markdown leaked repo root %q: %s", repo, markdown)
	}
}
