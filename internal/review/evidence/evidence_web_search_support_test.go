package evidence

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"time"

	"github.com/susugadx/xelyon-cli/internal/review/domain"
	"github.com/susugadx/xelyon-cli/internal/review/externaldoc"
)

func newReviewWebSearchEvidenceTestBundle() ReviewEvidenceBundle {
	return ReviewEvidenceBundle{
		TargetKind: domain.TargetCurrentChanges,
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

func newFetchedReviewExternalDocForWebSearchTest(content string, truncated bool) externaldoc.Evidence {
	return externaldoc.Evidence{
		FetchedAt:   time.Date(2026, time.May, 31, 0, 0, 0, 0, time.UTC),
		ContentHash: "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Truncated:   truncated,
		Snippets: []externaldoc.SnippetEvidence{
			{
				SnippetID:   "placeholder",
				Content:     content,
				ContentHash: "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
				Truncated:   truncated,
			},
		},
	}
}

func newReviewExternalSupportDocForEvidenceTest(docID string, credibility externaldoc.SourceCredibility, content string) externaldoc.Evidence {
	return externaldoc.Evidence{
		DocID:             docID,
		URL:               "https://platform.openai.com/docs/" + docID,
		SourceDomain:      "platform.openai.com",
		SourceCredibility: credibility,
		FetchedAt:         time.Date(2026, time.May, 31, 0, 0, 0, 0, time.UTC),
		ContentHash:       reviewExternalSupportHashForEvidenceTest("doc:" + docID),
		Snippets: []externaldoc.SnippetEvidence{
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

type fakeReviewWebSearchRunner struct {
	calls   int
	queries []string
	result  externaldoc.WebSearchQueryResult
	err     error
}

func (f *fakeReviewWebSearchRunner) SearchReviewWeb(_ context.Context, query string, _ int) (externaldoc.WebSearchQueryResult, error) {
	f.calls++
	f.queries = append(f.queries, query)
	return f.result, f.err
}

type fakeReviewExternalDocFetcher struct {
	calls    int
	requests []externaldoc.FetchRequest
	doc      externaldoc.Evidence
}

func (f *fakeReviewExternalDocFetcher) FetchExternalDoc(_ context.Context, req externaldoc.FetchRequest) externaldoc.Evidence {
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

func reviewExternalDocFocusTermsByTermForTest(terms []externaldoc.FocusTerm) map[string]string {
	result := make(map[string]string, len(terms))
	for _, term := range terms {
		result[term.Term] = term.Reason
	}
	return result
}
