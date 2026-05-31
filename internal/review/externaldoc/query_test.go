package externaldoc

import (
	"strings"
	"testing"
)

func TestBuildSearchQueryCandidatesRequireOfficialDocumentationAndFocus(t *testing.T) {
	got := BuildSearchQueryCandidates(newSearchQueryPlanningInputForTest("+ Tools: []map[string]any{{\"type\":\"web_search\"}}", []string{"web_search"}))

	if len(got) == 0 {
		t.Fatal("candidates empty, want focused official documentation query")
	}
	if got[0].Query != "OpenAI API web_search official documentation" {
		t.Fatalf("query = %q, want official documentation query", got[0].Query)
	}
	for _, candidate := range got {
		if !strings.Contains(candidate.Query, "official documentation") {
			t.Fatalf("query = %q, want official documentation phrase", candidate.Query)
		}
		if strings.HasSuffix(candidate.Query, " API official documentation") {
			t.Fatalf("query = %q, should not use generic fallback API focus", candidate.Query)
		}
	}
}

func TestBuildSearchQueryCandidatesSkipsSubjectOnlyAndGenericOnlyQueries(t *testing.T) {
	got := BuildSearchQueryCandidates(newSearchQueryPlanningInputForTest("+ endpoint := openaiBaseURL", []string{"API", "configuration"}))

	if len(got) != 0 {
		t.Fatalf("candidates = %#v, want none without concrete focus", got)
	}
}

func TestBuildSearchQueryCandidatesUsesCodeishGenericFocus(t *testing.T) {
	got := BuildSearchQueryCandidates(newSearchQueryPlanningInputForTest("+ response_format := schemaName", []string{"configuration", "response_format"}))

	if len(got) == 0 {
		t.Fatal("candidates empty, want code-ish generic focus query")
	}
	if got[0].Query != "OpenAI API response_format official documentation" {
		t.Fatalf("query = %q, want code-ish focus query", got[0].Query)
	}
}

func TestBuildFetchRequestMapsSearchHintsAndFocusTerms(t *testing.T) {
	got := BuildFetchRequest(
		SearchQueryCandidate{
			Query:   "OpenAI API web_search official documentation",
			Reason:  "changed external contract token: OpenAI API / web_search",
			Subject: "OpenAI API",
			Focus:   "web_search",
		},
		WebSearchEvidenceResult{
			Title:   "OpenAI Responses API previous_response_id guide",
			URL:     "https://docs.example.test/responses",
			Snippet: "Use tool_choice with text/event-stream.",
		},
		[]string{"web_search", "generic_safe_token"},
		"external-doc-1",
	)

	if got.URL != "https://docs.example.test/responses" {
		t.Fatalf("URL = %q, want search result URL", got.URL)
	}
	if got.DocID != "external-doc-1" {
		t.Fatalf("DocID = %q, want external-doc-1", got.DocID)
	}
	if got.SearchResultTitle != "OpenAI Responses API previous_response_id guide" {
		t.Fatalf("SearchResultTitle = %q, want search result title", got.SearchResultTitle)
	}
	if got.QuerySubjectHint != "OpenAI API" {
		t.Fatalf("QuerySubjectHint = %q, want query subject", got.QuerySubjectHint)
	}
	terms := reviewExternalDocFocusTermsByTermForTest(got.FocusTerms)
	for _, want := range []string{
		"OpenAI API",
		"web_search",
		"previous_response_id",
		"tool_choice",
		"text/event-stream",
		"generic_safe_token",
	} {
		if _, ok := terms[want]; !ok {
			t.Fatalf("focus terms = %#v, want %q", got.FocusTerms, want)
		}
	}
}

func newSearchQueryPlanningInputForTest(diff string, genericTokens []string) SearchQueryPlanningInput {
	return SearchQueryPlanningInput{
		CorpusParts: []string{
			"internal/api/providers/openai/client.go",
			diff,
		},
		GenericImpactTokens: genericTokens,
	}
}
