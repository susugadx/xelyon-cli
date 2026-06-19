package evidence

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"slices"
	"time"

	"github.com/susugadx/xelyon-cli/internal/review/externaldoc"
)

func newReviewWebSearchEvidenceTestBundle() ReviewEvidenceBundle {
	return ReviewEvidenceBundle{
		TargetKind: TargetCurrentChanges,
		RepoRoot:   "/tmp/repo",
		CWD:        "/tmp/repo",
		ChangedFiles: []ReviewChangedFile{
			{Path: "internal/api/providers/openai/web_search.go", Status: "M", Unstaged: true},
		},
		Diffs: []ReviewDiffEvidence{
			{
				Source: "unstaged",
				Stat:   "internal/api/providers/openai/web_search.go | 2 +",
				Diff:   "+ Tools: []map[string]any{{\"type\":\"web_search\"}}",
			},
		},
		Inventory: ReviewChangeInventory{
			Production: []string{"internal/api/providers/openai/web_search.go"},
		},
		GenericImpactCandidates: ReviewGenericImpactCandidates{
			Tokens: []string{"web_search"},
		},
		Limits: DefaultReviewEvidenceLimits(),
	}
}

func newFetchedReviewExternalDocForWebSearchTest(content string, truncated bool) ReviewExternalDocEvidence {
	return ReviewExternalDocEvidence{
		FetchedAt:   time.Date(2026, time.May, 31, 0, 0, 0, 0, time.UTC),
		ContentHash: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Truncated:   truncated,
		Snippets: []ReviewExternalDocSnippetEvidence{
			{
				SnippetID:   "placeholder",
				Content:     content,
				ContentHash: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
				Truncated:   truncated,
			},
		},
	}
}

func newReviewExternalSupportDocForEvidenceTest(docID string, credibility externaldoc.SourceCredibility, content string) ReviewExternalDocEvidence {
	return ReviewExternalDocEvidence{
		DocID:             docID,
		URL:               "https://platform.openai.com/docs/" + docID,
		SourceDomain:      "platform.openai.com",
		SourceCredibility: credibility,
		FetchedAt:         time.Date(2026, time.May, 31, 0, 0, 0, 0, time.UTC),
		ContentHash:       reviewExternalSupportHashForEvidenceTest("doc:" + docID),
		Snippets: []ReviewExternalDocSnippetEvidence{
			{
				SnippetID:   docID + "-snippet-1",
				Content:     content,
				ContentHash: reviewExternalSupportHashForEvidenceTest("snippet:" + content),
			},
		},
	}
}

func reviewExternalSupportHashForEvidenceTest(seed string) string {
	sum := sha256.Sum256([]byte(seed))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func cloneReviewWebSearchEvidenceForTest(evidence ReviewWebSearchEvidence) ReviewWebSearchEvidence {
	clone := evidence
	clone.Queries = append([]ReviewWebSearchEvidenceQuery(nil), evidence.Queries...)
	for i := range clone.Queries {
		clone.Queries[i].Results = append([]ReviewWebSearchEvidenceResult(nil), evidence.Queries[i].Results...)
	}
	clone.ExternalDocs = append([]ReviewExternalDocEvidence(nil), evidence.ExternalDocs...)
	for i := range clone.ExternalDocs {
		clone.ExternalDocs[i].Snippets = append([]ReviewExternalDocSnippetEvidence(nil), evidence.ExternalDocs[i].Snippets...)
	}
	return clone
}

type fakeReviewWebSearchRunner struct {
	calls   int
	queries []string
	result  ReviewWebSearchQueryResult
	err     error
}

func (f *fakeReviewWebSearchRunner) SearchReviewWeb(_ context.Context, query string, _ int) (ReviewWebSearchQueryResult, error) {
	f.calls++
	f.queries = append(f.queries, query)
	return f.result, f.err
}

type fakeReviewExternalDocFetcher struct {
	calls    int
	requests []ReviewExternalDocFetchRequest
	doc      ReviewExternalDocEvidence
}

func (f *fakeReviewExternalDocFetcher) FetchExternalDoc(_ context.Context, req ReviewExternalDocFetchRequest) ReviewExternalDocEvidence {
	f.calls++
	f.requests = append(f.requests, req)
	doc := f.doc
	doc.DocID = req.DocID
	doc.URL = req.URL
	doc.SourceDomain = "docs.example.test"
	for i := range doc.Snippets {
		doc.Snippets[i].SnippetID = req.DocID + "-snippet-1"
	}
	return doc
}

func reviewPressureSignalsContain(signals []ReviewPressureSignalInput, want string) bool {
	return slices.ContainsFunc(signals, func(signal ReviewPressureSignalInput) bool {
		return signal.Signal == want
	})
}

func reviewExternalDocFocusTermsByTermForTest(terms []ReviewExternalDocFocusTerm) map[string]string {
	result := make(map[string]string, len(terms))
	for _, term := range terms {
		result[term.Term] = term.Reason
	}
	return result
}
