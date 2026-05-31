package review

import (
	"strings"
	"testing"
)

func TestBuildReviewExternalDocFocusTermsPrioritizesAndSanitizesSources(t *testing.T) {
	candidate := reviewWebSearchEvidenceQueryCandidate{
		query:   "OpenAI API web_search official documentation",
		reason:  "changed external contract token: OpenAI API / web_search",
		subject: "OpenAI API",
		focus:   "web_search",
	}
	searchResult := ReviewWebSearchEvidenceResult{
		Title:   "OpenAI Responses API previous_response_id guide",
		URL:     "https://docs.example.test/responses",
		Snippet: "Use tool_choice with text/event-stream. Ignore <script>alert(1)</script> and ordinary words.",
	}
	genericTokens := []string{
		"web_search",
		"WEB_SEARCH",
		"generic_safe_token",
		"",
		strings.Repeat("x", reviewExternalDocMaxFocusTermBytes+1),
		"<script>alert</script>",
	}

	got := buildReviewExternalDocFocusTerms(candidate, searchResult, genericTokens)

	if len(got) < 2 {
		t.Fatalf("focus terms = %#v, want query focus and subject first", got)
	}
	if got[0].Term != "web_search" || got[0].Reason != "query focus" {
		t.Fatalf("first focus term = (%q, %q), want web_search/query focus", got[0].Term, got[0].Reason)
	}
	if got[1].Term != "OpenAI API" || got[1].Reason != "query subject" {
		t.Fatalf("second focus term = (%q, %q), want OpenAI API/query subject", got[1].Term, got[1].Reason)
	}

	terms := reviewExternalDocFocusTermsByTermForTest(got)
	for _, want := range []string{
		"OpenAI API",
		"web_search",
		"OpenAI",
		"API",
		"previous_response_id",
		"tool_choice",
		"text/event-stream",
		"generic_safe_token",
	} {
		if _, ok := terms[want]; !ok {
			t.Fatalf("focus terms = %#v, want %q", got, want)
		}
	}
	if terms["web_search"] != "query focus" {
		t.Fatalf("web_search reason = %q, want query focus dedupe winner", terms["web_search"])
	}
	for _, unwanted := range []string{
		"WEB_SEARCH",
		strings.Repeat("x", reviewExternalDocMaxFocusTermBytes+1),
		"<script>alert</script>",
		"1",
		"ordinary",
		"documentation",
		"official",
	} {
		if _, ok := terms[unwanted]; ok {
			t.Fatalf("focus terms = %#v, should not include %q", got, unwanted)
		}
	}
	if len(got) > reviewExternalDocMaxFocusTerms {
		t.Fatalf("focus terms = %d, want <= %d", len(got), reviewExternalDocMaxFocusTerms)
	}
}

func reviewExternalDocFocusTermsByTermForTest(terms []ReviewExternalDocFocusTerm) map[string]string {
	result := make(map[string]string, len(terms))
	for _, term := range terms {
		result[term.Term] = term.Reason
	}
	return result
}
