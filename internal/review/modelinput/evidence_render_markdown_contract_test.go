package modelinput

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/review/domain"
	reviewevidence "github.com/susugadx/xelyon-cli/internal/review/evidence"
)

func TestRenderReviewEvidenceMarkdownQuotesControlledSubsectionTitle(t *testing.T) {
	repo := filepath.Clean(t.TempDir())
	bundle := reviewevidence.ReviewEvidenceBundle{
		TargetKind: domain.TargetCurrentChanges,
		RepoRoot:   repo,
		UntrackedFiles: []reviewevidence.ReviewUntrackedFile{{
			Path:     "notes\n## limits",
			Snapshot: "body\n",
		}},
	}

	markdown := RenderReviewEvidenceMarkdown(bundle)

	if !strings.Contains(markdown, "### \"untracked file: notes\\n## limits\"\n") {
		t.Fatalf("markdown = %q, want quoted untracked subsection heading", markdown)
	}
	if strings.Contains(markdown, "### untracked file: notes\n## limits\n") {
		t.Fatalf("markdown contains raw controlled subsection heading: %q", markdown)
	}
	if got := strings.Count(markdown, "\n## limits\n"); got != 1 {
		t.Fatalf("limits section count = %d, want 1:\n%s", got, markdown)
	}
}

func TestRenderReviewEvidenceMarkdownExpandsFenceForBackticks(t *testing.T) {
	bundle := newReviewEvidenceRenderTestBundle(t)
	bundle.RuleFiles = []reviewevidence.ReviewRuleFileEvidence{{
		Path:    "AGENTS.md",
		Content: "rule ````` content",
	}}
	bundle.Diffs = []reviewevidence.ReviewDiffEvidence{{
		Source: "unstaged",
		Diff:   "+diff ````` body",
	}}
	bundle.UntrackedFiles = []reviewevidence.ReviewUntrackedFile{{
		Path:     "notes.md",
		Snapshot: "untracked ````` snapshot",
	}}

	markdown := RenderReviewEvidenceMarkdown(bundle)

	for _, want := range []string{
		"``````text\nrule ````` content\n``````",
		"``````diff\n+diff ````` body\n``````",
		"``````text\nuntracked ````` snapshot\n``````",
	} {
		if !strings.Contains(markdown, want) {
			t.Fatalf("markdown = %q, want expanded fence block %q", markdown, want)
		}
	}
}
