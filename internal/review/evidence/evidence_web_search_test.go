package evidence

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/review/externaldoc"
)

func TestReviewWebSearchEvidenceCollectorDisabledDoesNotCallDependencies(t *testing.T) {
	searcher := &fakeReviewWebSearchRunner{}
	fetcher := &fakeReviewExternalDocFetcher{}
	collector := NewReviewWebSearchEvidenceCollector(ReviewWebSearchEvidenceCollectorOptions{
		Enabled:  false,
		Searcher: searcher,
		Fetcher:  fetcher,
	})

	got := collector.CollectWebSearchEvidence(context.Background(), newReviewWebSearchEvidenceTestBundle())

	if got.Enabled {
		t.Fatal("Enabled = true, want false")
	}
	if searcher.calls != 0 {
		t.Fatalf("searcher calls = %d, want 0", searcher.calls)
	}
	if fetcher.calls != 0 {
		t.Fatalf("fetcher calls = %d, want 0", fetcher.calls)
	}
}

func TestReviewWebSearchEvidenceCollectorSearchesAndFetchesBoundedResults(t *testing.T) {
	searcher := &fakeReviewWebSearchRunner{
		result: externaldoc.WebSearchQueryResult{
			Provider: "gemini",
			Results: []externaldoc.WebSearchEvidenceResult{
				{Title: "one", URL: "https://docs.example.test/one", SourceDomain: "docs.example.test"},
				{Title: "two", URL: "https://docs.example.test/two", SourceDomain: "docs.example.test"},
				{Title: "three", URL: "https://docs.example.test/three", SourceDomain: "docs.example.test"},
			},
		},
	}
	fetcher := &fakeReviewExternalDocFetcher{
		doc: newFetchedReviewExternalDocForWebSearchTest("external spec", false),
	}
	collector := NewReviewWebSearchEvidenceCollector(ReviewWebSearchEvidenceCollectorOptions{
		Enabled:            true,
		MaxQueries:         1,
		MaxResultsPerQuery: 2,
		Searcher:           searcher,
		Fetcher:            fetcher,
	})

	got := collector.CollectWebSearchEvidence(context.Background(), newReviewWebSearchEvidenceTestBundle())

	if !got.Enabled {
		t.Fatal("Enabled = false, want true")
	}
	if got.Provider != "gemini" {
		t.Fatalf("Provider = %q, want gemini", got.Provider)
	}
	if searcher.calls != 1 {
		t.Fatalf("searcher calls = %d, want 1", searcher.calls)
	}
	if fetcher.calls != 2 {
		t.Fatalf("fetcher calls = %d, want 2", fetcher.calls)
	}
	if len(got.Queries) != 1 || len(got.Queries[0].Results) != 2 {
		t.Fatalf("queries = %#v, want one query with two bounded results", got.Queries)
	}
	if got.Inconclusive {
		t.Fatal("Inconclusive = true, want false with fetched snippets")
	}
	if !got.Truncated {
		t.Fatal("Truncated = false, want true when search results exceed max_results_per_query")
	}
	if got.Queries[0].Query != "OpenAI API web_search official documentation" {
		t.Fatalf("query = %q, want OpenAI/web_search official docs query", got.Queries[0].Query)
	}
	if !strings.Contains(got.Queries[0].Reason, "intent=api_docs") || !strings.Contains(got.Queries[0].Reason, "expected_source_type=api_reference") {
		t.Fatalf("query reason = %q, want intent metadata", got.Queries[0].Reason)
	}
}

func TestReviewWebSearchEvidenceCollectorPassesSafeFocusTermsToFetcher(t *testing.T) {
	searcher := &fakeReviewWebSearchRunner{
		result: externaldoc.WebSearchQueryResult{
			Provider: "gemini",
			Results: []externaldoc.WebSearchEvidenceResult{
				{
					Title:   "OpenAI Responses API previous_response_id guide",
					URL:     "https://docs.example.test/responses",
					Snippet: "Use tool_choice with text/event-stream. Ignore <script>alert(1)</script> and ordinary words.",
				},
			},
		},
	}
	fetcher := &fakeReviewExternalDocFetcher{
		doc: newFetchedReviewExternalDocForWebSearchTest("external spec", false),
	}
	bundle := newReviewWebSearchEvidenceTestBundle()
	bundle.GenericImpactCandidates.Tokens = []string{
		"web_search",
		"generic_safe_token",
	}
	collector := NewReviewWebSearchEvidenceCollector(ReviewWebSearchEvidenceCollectorOptions{
		Enabled:            true,
		MaxQueries:         1,
		MaxResultsPerQuery: 1,
		Searcher:           searcher,
		Fetcher:            fetcher,
	})

	_ = collector.CollectWebSearchEvidence(context.Background(), bundle)

	if len(fetcher.requests) != 1 {
		t.Fatalf("fetcher requests = %d, want 1", len(fetcher.requests))
	}
	if fetcher.requests[0].SearchResultTitle != "OpenAI Responses API previous_response_id guide" {
		t.Fatalf("SearchResultTitle = %q, want search result title", fetcher.requests[0].SearchResultTitle)
	}
	if fetcher.requests[0].QuerySubjectHint != "OpenAI API" {
		t.Fatalf("QuerySubjectHint = %q, want query subject", fetcher.requests[0].QuerySubjectHint)
	}
	terms := reviewExternalDocFocusTermsByTermForTest(fetcher.requests[0].FocusTerms)
	for _, want := range []string{
		"OpenAI API",
		"web_search",
		"previous_response_id",
		"generic_safe_token",
	} {
		if _, ok := terms[want]; !ok {
			t.Fatalf("focus terms = %#v, want %q", fetcher.requests[0].FocusTerms, want)
		}
	}
}

func TestReviewWebSearchEvidenceCollectorPropagatesFetcherTruncation(t *testing.T) {
	searcher := &fakeReviewWebSearchRunner{
		result: externaldoc.WebSearchQueryResult{
			Provider: "gemini",
			Results: []externaldoc.WebSearchEvidenceResult{
				{Title: "one", URL: "https://docs.example.test/one", SourceDomain: "docs.example.test"},
			},
		},
	}
	fetcher := &fakeReviewExternalDocFetcher{
		doc: newFetchedReviewExternalDocForWebSearchTest("truncated external spec", true),
	}
	collector := NewReviewWebSearchEvidenceCollector(ReviewWebSearchEvidenceCollectorOptions{
		Enabled:            true,
		MaxQueries:         1,
		MaxResultsPerQuery: 1,
		Searcher:           searcher,
		Fetcher:            fetcher,
	})

	got := collector.CollectWebSearchEvidence(context.Background(), newReviewWebSearchEvidenceTestBundle())

	if !got.Truncated {
		t.Fatal("Truncated = false, want true when fetched external_doc is truncated")
	}
}

func TestReviewWebSearchEvidenceCollectorPropagatesSearchTruncation(t *testing.T) {
	searcher := &fakeReviewWebSearchRunner{
		result: externaldoc.WebSearchQueryResult{
			Provider:  "gemini",
			Truncated: true,
			Results: []externaldoc.WebSearchEvidenceResult{
				{Title: "one", URL: "https://docs.example.test/one", SourceDomain: "docs.example.test"},
				{Title: "two", URL: "https://docs.example.test/two", SourceDomain: "docs.example.test"},
			},
		},
	}
	fetcher := &fakeReviewExternalDocFetcher{
		doc: newFetchedReviewExternalDocForWebSearchTest("external spec", false),
	}
	collector := NewReviewWebSearchEvidenceCollector(ReviewWebSearchEvidenceCollectorOptions{
		Enabled:            true,
		MaxQueries:         1,
		MaxResultsPerQuery: 2,
		Searcher:           searcher,
		Fetcher:            fetcher,
	})

	got := collector.CollectWebSearchEvidence(context.Background(), newReviewWebSearchEvidenceTestBundle())

	if !got.Truncated {
		t.Fatal("Truncated = false, want true when search provider reports truncation")
	}
	if len(got.Queries) != 1 || len(got.Queries[0].Results) != 2 {
		t.Fatalf("queries = %#v, want max results without local truncation", got.Queries)
	}
}

func TestReviewWebSearchEvidenceCollectorSearchFailureIsEvidenceNotRunnerError(t *testing.T) {
	searcher := &fakeReviewWebSearchRunner{err: errors.New("search failed")}
	collector := NewReviewWebSearchEvidenceCollector(ReviewWebSearchEvidenceCollectorOptions{
		Enabled:  true,
		Searcher: searcher,
		Fetcher:  &fakeReviewExternalDocFetcher{},
	})

	got := collector.CollectWebSearchEvidence(context.Background(), newReviewWebSearchEvidenceTestBundle())

	if got.Error == "" || !strings.Contains(got.Error, "search failed") {
		t.Fatalf("Error = %q, want search failure", got.Error)
	}
	if !got.Inconclusive {
		t.Fatal("Inconclusive = false, want true when search fails")
	}
}
