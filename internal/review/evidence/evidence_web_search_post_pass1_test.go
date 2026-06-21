package evidence

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/review/domain"
	"github.com/susugadx/xelyon-cli/internal/review/externaldoc"
	reviewprobeplan "github.com/susugadx/xelyon-cli/internal/review/probeplan"
	"github.com/susugadx/xelyon-cli/internal/review/report"
)

func TestReviewWebSearchEvidenceCollectorPostPass1SearchesPlanDerivedQuery(t *testing.T) {
	searcher := &fakeReviewWebSearchRunner{
		result: externaldoc.WebSearchQueryResult{
			Provider: "gemini",
			Results: []externaldoc.WebSearchEvidenceResult{
				{Title: "OAuth redirect URI spec", URL: "https://docs.example.test/oauth", SourceDomain: "docs.example.test"},
			},
		},
	}
	fetcher := &fakeReviewExternalDocFetcher{
		doc: newFetchedReviewExternalDocForWebSearchTest("OAuth redirect URI external spec", false),
	}
	bundle := newReviewWebSearchEvidenceTestBundle()
	bundle.WebSearchEvidence = externaldoc.WebSearchEvidence{
		Enabled: true,
		Queries: []externaldoc.WebSearchEvidenceQuery{
			{Query: "OpenAI API web_search official documentation", Reason: "pre-pass1"},
		},
		ExternalDocs: []externaldoc.Evidence{
			newReviewExternalSupportDocForEvidenceTest("external-doc-1", externaldoc.SourceCredibilityOfficialCandidate, "pre-pass1 snippet"),
		},
	}
	collector := NewReviewWebSearchEvidenceCollector(ReviewWebSearchEvidenceCollectorOptions{
		Enabled:            true,
		MaxQueries:         3,
		MaxResultsPerQuery: 1,
		Searcher:           searcher,
		Fetcher:            fetcher,
	})

	got := collector.CollectPostPass1WebSearchEvidence(context.Background(), bundle, newReviewWebSearchPostPass1OAuthPlanForTest())

	if searcher.calls != 1 {
		t.Fatalf("searcher calls = %d, want 1", searcher.calls)
	}
	if len(got.Queries) != 2 {
		t.Fatalf("queries = %#v, want pre query plus post query", got.Queries)
	}
	if got.Queries[1].Query != "OAuth 2.0 redirect URI specification" {
		t.Fatalf("post query = %q, want OAuth spec query", got.Queries[1].Query)
	}
	if !strings.Contains(got.Queries[1].Reason, "intent=spec") || !strings.Contains(got.Queries[1].Reason, "expected_source_type=technical_specification") {
		t.Fatalf("post query reason = %q, want spec metadata", got.Queries[1].Reason)
	}
	if len(got.ExternalDocs) != 2 || got.ExternalDocs[1].DocID != "external-doc-2" {
		t.Fatalf("external docs = %#v, want post doc with non-conflicting external-doc-2", got.ExternalDocs)
	}
	if got.Inconclusive {
		t.Fatal("Inconclusive = true, want false with merged snippets")
	}
}

func TestReviewWebSearchEvidenceCollectorPostPass1SkipsDuplicateQuery(t *testing.T) {
	searcher := &fakeReviewWebSearchRunner{}
	bundle := newReviewWebSearchEvidenceTestBundle()
	bundle.WebSearchEvidence = externaldoc.WebSearchEvidence{
		Enabled: true,
		Queries: []externaldoc.WebSearchEvidenceQuery{
			{Query: "OAuth 2.0 redirect URI specification", Reason: "pre-pass1"},
		},
	}
	collector := NewReviewWebSearchEvidenceCollector(ReviewWebSearchEvidenceCollectorOptions{
		Enabled:            true,
		MaxQueries:         3,
		MaxResultsPerQuery: 1,
		Searcher:           searcher,
		Fetcher:            &fakeReviewExternalDocFetcher{},
	})

	got := collector.CollectPostPass1WebSearchEvidence(context.Background(), bundle, newReviewWebSearchPostPass1OAuthPlanForTest())

	if searcher.calls != 0 {
		t.Fatalf("searcher calls = %d, want 0 for duplicate post query", searcher.calls)
	}
	if len(got.Queries) != 1 {
		t.Fatalf("queries = %#v, want only existing duplicate query", got.Queries)
	}
}

func TestReviewWebSearchEvidenceCollectorPostPass1RespectsTotalQueryBudget(t *testing.T) {
	searcher := &fakeReviewWebSearchRunner{}
	bundle := newReviewWebSearchEvidenceTestBundle()
	bundle.WebSearchEvidence = externaldoc.WebSearchEvidence{
		Enabled: true,
		Queries: []externaldoc.WebSearchEvidenceQuery{
			{Query: "OpenAI API web_search official documentation", Reason: "pre-pass1"},
		},
	}
	collector := NewReviewWebSearchEvidenceCollector(ReviewWebSearchEvidenceCollectorOptions{
		Enabled:            true,
		MaxQueries:         1,
		MaxResultsPerQuery: 1,
		Searcher:           searcher,
		Fetcher:            &fakeReviewExternalDocFetcher{},
	})

	got := collector.CollectPostPass1WebSearchEvidence(context.Background(), bundle, newReviewWebSearchPostPass1OAuthPlanForTest())

	if searcher.calls != 0 {
		t.Fatalf("searcher calls = %d, want 0 when pre-pass1 exhausted budget", searcher.calls)
	}
	if !got.Truncated {
		t.Fatal("Truncated = false, want true when post-pass1 candidates exceed remaining budget")
	}
	if len(got.Queries) != 1 {
		t.Fatalf("queries = %#v, want pre-pass1 query only", got.Queries)
	}
}

func TestReviewWebSearchEvidenceCollectorPostPass1ErrorPreservesMergedSignals(t *testing.T) {
	searcher := &fakeReviewWebSearchRunner{err: errors.New("post search failed")}
	bundle := newReviewWebSearchEvidenceTestBundle()
	bundle.WebSearchEvidence = externaldoc.WebSearchEvidence{
		Enabled: true,
		Error:   "pre fetch failed",
		ExternalDocs: []externaldoc.Evidence{
			newReviewExternalSupportDocForEvidenceTest("external-doc-1", externaldoc.SourceCredibilityUnknown, "pre-pass1 snippet"),
		},
	}
	collector := NewReviewWebSearchEvidenceCollector(ReviewWebSearchEvidenceCollectorOptions{
		Enabled:            true,
		MaxQueries:         3,
		MaxResultsPerQuery: 1,
		Searcher:           searcher,
		Fetcher:            &fakeReviewExternalDocFetcher{},
	})

	got := collector.CollectPostPass1WebSearchEvidence(context.Background(), bundle, newReviewWebSearchPostPass1OAuthPlanForTest())

	if !strings.Contains(got.Error, "pre fetch failed") || !strings.Contains(got.Error, "post search failed") {
		t.Fatalf("Error = %q, want pre and post errors preserved", got.Error)
	}
	if got.Inconclusive {
		t.Fatal("Inconclusive = true, want false because merged docs have citation-capable snippet")
	}
	if len(got.ExternalDocs) != 1 {
		t.Fatalf("external docs = %#v, want existing docs preserved", got.ExternalDocs)
	}
}

func newReviewWebSearchPostPass1OAuthPlanForTest() reviewprobeplan.ReviewProbePlan {
	return reviewprobeplan.ReviewProbePlan{
		SchemaVersion: reviewprobeplan.ReviewProbePlanSchemaVersionV2,
		TargetKind:    domain.TargetCurrentChanges,
		ImpactSurfaces: []reviewprobeplan.ReviewProbeImpactSurface{
			{
				ID:              "surface-oauth",
				Summary:         "OAuth redirect URI validation changed.",
				Category:        reviewprobeplan.ReviewProbeImpactSurfaceValidator,
				EvidenceSummary: "Diff mentions redirect_uri and token exchange.",
				Status:          reviewprobeplan.ReviewProbeImpactSurfaceUnverified,
				Reason:          "External OAuth 2.0 specification should be checked.",
			},
		},
		CandidateRisks: []reviewprobeplan.ReviewProbeCandidateRisk{
			{
				ID:                   "risk-oauth",
				Summary:              "OAuth flow could be accepted with a mismatched redirect URI.",
				Severity:             report.ReviewGroupSeverityMedium,
				SurfaceIDs:           []string{"surface-oauth"},
				VerificationStrategy: "Confirm redirect URI requirements against OAuth 2.0 specification.",
				Status:               reviewprobeplan.ReviewProbeCandidateRiskUnverified,
			},
		},
	}
}
