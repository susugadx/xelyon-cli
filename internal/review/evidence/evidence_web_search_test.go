package evidence

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

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
		result: ReviewWebSearchQueryResult{
			Provider: "gemini",
			Results: []ReviewWebSearchEvidenceResult{
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
}

func TestReviewWebSearchEvidenceCollectorPassesSafeFocusTermsToFetcher(t *testing.T) {
	searcher := &fakeReviewWebSearchRunner{
		result: ReviewWebSearchQueryResult{
			Provider: "gemini",
			Results: []ReviewWebSearchEvidenceResult{
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
		result: ReviewWebSearchQueryResult{
			Provider: "gemini",
			Results: []ReviewWebSearchEvidenceResult{
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
		result: ReviewWebSearchQueryResult{
			Provider:  "gemini",
			Truncated: true,
			Results: []ReviewWebSearchEvidenceResult{
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

func TestReviewEvidenceMarkdownIncludesWebSearchEvidenceSection(t *testing.T) {
	bundle := newReviewWebSearchEvidenceTestBundle()
	bundle.WebSearchEvidence = ReviewWebSearchEvidence{
		Enabled:  true,
		Provider: "gemini",
		Queries:  []ReviewWebSearchEvidenceQuery{{Query: "OpenAI API web_search official documentation", Reason: "test"}},
		ExternalDocs: []ReviewExternalDocEvidence{
			{
				DocID:                   "external-doc-1",
				URL:                     "https://example.test/openai-reference",
				SourceDomain:            "example.test",
				SourceCredibility:       externaldoc.SourceCredibilityUnknown,
				SourceCredibilityReason: "unknown: source domain does not match trusted domains for the query subject",
				Snippets: []ReviewExternalDocSnippetEvidence{
					{
						SnippetID:   "external-doc-1-snippet-1",
						Content:     "OpenAI API request example.",
						ContentHash: "hash-1",
					},
				},
			},
		},
	}

	got := RenderReviewEvidenceMarkdown(bundle)

	for _, want := range []string{
		"## review web search evidence",
		`"provider": "gemini"`,
		`"OpenAI API web_search official documentation"`,
		`"source_credibility": "unknown"`,
		`"source_credibility_reason": "unknown: source domain does not match trusted domains for the query subject"`,
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("markdown missing %q:\n%s", want, got)
		}
	}
}

func TestReviewPressureSignalsIncludeWebSearchEvidenceStates(t *testing.T) {
	input := BuildReviewEvidenceModelInput(newReviewWebSearchEvidenceTestBundle())
	disabledSignals := BuildReviewPressureSignalInputs(input)
	if !reviewPressureSignalsContain(disabledSignals, "web_search_evidence_disabled_for_external_contract_change") {
		t.Fatalf("signals = %#v, want disabled external contract signal", disabledSignals)
	}

	bundle := newReviewWebSearchEvidenceTestBundle()
	bundle.WebSearchEvidence = ReviewWebSearchEvidence{
		Enabled:      true,
		Error:        "fetch failed",
		Truncated:    true,
		Inconclusive: true,
	}
	enabledSignals := BuildReviewPressureSignalInputs(BuildReviewEvidenceModelInput(bundle))
	for _, signal := range []string{
		"web_search_evidence_failed",
		"web_search_evidence_truncated",
		"web_search_evidence_inconclusive",
	} {
		if !reviewPressureSignalsContain(enabledSignals, signal) {
			t.Fatalf("signals = %#v, want %s", enabledSignals, signal)
		}
	}
}

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

type fakeReviewWebSearchRunner struct {
	calls  int
	result ReviewWebSearchQueryResult
	err    error
}

func (f *fakeReviewWebSearchRunner) SearchReviewWeb(_ context.Context, _ string, _ int) (ReviewWebSearchQueryResult, error) {
	f.calls++
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
