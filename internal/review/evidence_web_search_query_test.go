package review

import (
	"strings"
	"testing"
)

func TestBuildReviewWebSearchEvidenceQueryCandidatesRequireOfficialDocumentationAndFocus(t *testing.T) {
	got := buildReviewWebSearchEvidenceQueryCandidates(newReviewWebSearchEvidenceTestBundle())

	if len(got) == 0 {
		t.Fatal("candidates empty, want focused official documentation query")
	}
	if got[0].query != "OpenAI API web_search official documentation" {
		t.Fatalf("query = %q, want official documentation query", got[0].query)
	}
	for _, candidate := range got {
		if !strings.Contains(candidate.query, "official documentation") {
			t.Fatalf("query = %q, want official documentation phrase", candidate.query)
		}
		if strings.HasSuffix(candidate.query, " API official documentation") {
			t.Fatalf("query = %q, should not use generic fallback API focus", candidate.query)
		}
	}
}

func TestBuildReviewWebSearchEvidenceQueryCandidatesSkipsSubjectOnlyAndGenericOnlyQueries(t *testing.T) {
	got := buildReviewWebSearchEvidenceQueryCandidates(newReviewWebSearchEvidenceQueryCandidateBundleForTest("+ endpoint := openaiBaseURL", []string{"API", "configuration"}))

	if len(got) != 0 {
		t.Fatalf("candidates = %#v, want none without concrete focus", got)
	}
}

func TestBuildReviewWebSearchEvidenceQueryCandidatesUsesCodeishGenericFocus(t *testing.T) {
	got := buildReviewWebSearchEvidenceQueryCandidates(newReviewWebSearchEvidenceQueryCandidateBundleForTest("+ response_format := schemaName", []string{"configuration", "response_format"}))

	if len(got) == 0 {
		t.Fatal("candidates empty, want code-ish generic focus query")
	}
	if got[0].query != "OpenAI API response_format official documentation" {
		t.Fatalf("query = %q, want code-ish focus query", got[0].query)
	}
}

func newReviewWebSearchEvidenceQueryCandidateBundleForTest(diff string, genericTokens []string) ReviewEvidenceBundle {
	return ReviewEvidenceBundle{
		ChangedFiles: []ReviewChangedFile{
			{Path: "internal/api/providers/openai/client.go", Status: "M", Unstaged: true},
		},
		Diffs: []ReviewDiffEvidence{{Diff: diff}},
		GenericImpactCandidates: ReviewGenericImpactCandidates{
			Tokens: genericTokens,
		},
	}
}
