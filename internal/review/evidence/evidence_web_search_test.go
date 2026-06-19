package evidence

import (
	"context"
	"errors"
	"reflect"
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
	if !strings.Contains(got.Queries[0].Reason, "intent=api_docs") || !strings.Contains(got.Queries[0].Reason, "expected_source_type=api_reference") {
		t.Fatalf("query reason = %q, want intent metadata", got.Queries[0].Reason)
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

func TestBuildReviewEvidenceModelInputIncludesExternalSupportSummary(t *testing.T) {
	bundle := newReviewWebSearchEvidenceTestBundle()
	bundle.WebSearchEvidence = ReviewWebSearchEvidence{
		Enabled: true,
		ExternalDocs: []ReviewExternalDocEvidence{
			newReviewExternalSupportDocForEvidenceTest("external-doc-1", externaldoc.SourceCredibilityOfficialCandidate, "first official snippet"),
			newReviewExternalSupportDocForEvidenceTest("external-doc-2", externaldoc.SourceCredibilityOfficialCandidate, "second official snippet"),
		},
	}

	input := BuildReviewEvidenceModelInput(bundle)

	if input.ExternalSupport.Level != externaldoc.ExternalSupportLevelAdequate {
		t.Fatalf("ExternalSupport.Level = %q, want adequate", input.ExternalSupport.Level)
	}
	if !input.ExternalSupport.OfficialConfirmation {
		t.Fatal("ExternalSupport.OfficialConfirmation = false, want true")
	}
	if input.ExternalSupport.OfficialCandidateCitationCapableDocCount != 2 {
		t.Fatalf("OfficialCandidateCitationCapableDocCount = %d, want 2", input.ExternalSupport.OfficialCandidateCitationCapableDocCount)
	}
	if input.ExternalSupport.OfficialCandidateUniqueCitationCapableSourceCount != 2 {
		t.Fatalf("OfficialCandidateUniqueCitationCapableSourceCount = %d, want 2", input.ExternalSupport.OfficialCandidateUniqueCitationCapableSourceCount)
	}
}

func TestReviewEvidenceMarkdownIncludesExternalSupportSummary(t *testing.T) {
	bundle := newReviewWebSearchEvidenceTestBundle()
	bundle.WebSearchEvidence = ReviewWebSearchEvidence{
		Enabled: true,
		ExternalDocs: []ReviewExternalDocEvidence{
			newReviewExternalSupportDocForEvidenceTest("external-doc-1", externaldoc.SourceCredibilityOfficialCandidate, "first official snippet"),
			newReviewExternalSupportDocForEvidenceTest("external-doc-2", externaldoc.SourceCredibilityOfficialCandidate, "second official snippet"),
		},
	}

	got := RenderReviewEvidenceMarkdown(bundle)

	for _, want := range []string{
		"## external support summary",
		`"level": "adequate"`,
		`"citation_capable_doc_count": 2`,
		`"official_candidate_citation_capable_doc_count": 2`,
		`"official_candidate_unique_citation_capable_source_count": 2`,
		`"official_confirmation": true`,
		"## review web search evidence",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("markdown missing %q:\n%s", want, got)
		}
	}
}

func TestExternalSupportSummaryDoesNotMutateWebSearchEvidence(t *testing.T) {
	bundle := newReviewWebSearchEvidenceTestBundle()
	bundle.WebSearchEvidence = ReviewWebSearchEvidence{
		Enabled: true,
		Queries: []ReviewWebSearchEvidenceQuery{
			{
				Query: "OpenAI API web_search official documentation",
				Results: []ReviewWebSearchEvidenceResult{
					{Title: "OpenAI docs", URL: "https://platform.openai.com/docs"},
				},
			},
		},
		ExternalDocs: []ReviewExternalDocEvidence{
			newReviewExternalSupportDocForEvidenceTest("external-doc-1", externaldoc.SourceCredibilityOfficialCandidate, "first official snippet"),
		},
	}
	original := cloneReviewWebSearchEvidenceForTest(bundle.WebSearchEvidence)

	_ = BuildReviewEvidenceModelInput(bundle)
	_ = RenderReviewEvidenceMarkdown(bundle)

	if !reflect.DeepEqual(bundle.WebSearchEvidence, original) {
		t.Fatalf("WebSearchEvidence mutated:\n got %#v\nwant %#v", bundle.WebSearchEvidence, original)
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
