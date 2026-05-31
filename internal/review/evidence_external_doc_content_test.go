package review

import (
	"strings"
	"testing"
)

func TestSanitizeReviewExternalDocTextRemovesHTMLNoise(t *testing.T) {
	body := `<html><head><style>.x{}</style><script>alert("x")</script></head><body><nav>navigation-leak previous_response_id</nav><h1>Spec</h1><p>` +
		strings.Repeat("External contract text ", 120) +
		`</p></body></html>`

	sanitized := sanitizeReviewExternalDocText([]byte(body), "text/html; charset=utf-8")
	snippets := buildReviewExternalDocSnippets("external-doc-1", sanitized, false, nil)

	if len(snippets) == 0 {
		t.Fatal("Snippets empty, want sanitized snippet")
	}
	for _, leaked := range []string{"<script", "alert", "navigation-leak"} {
		if strings.Contains(snippets[0].Content, leaked) {
			t.Fatalf("snippet leaked sanitized HTML content %q: %q", leaked, snippets[0].Content)
		}
	}
	if len(snippets[0].Content) > reviewExternalDocMaxSnippetBytes {
		t.Fatalf("snippet bytes = %d, want <= %d", len(snippets[0].Content), reviewExternalDocMaxSnippetBytes)
	}
	if got := reviewExternalDocContentHash(sanitized); !strings.HasPrefix(got, "sha256:") {
		t.Fatalf("ContentHash = %q, want sha256 hash", got)
	}
	if !strings.HasPrefix(snippets[0].ContentHash, "sha256:") {
		t.Fatalf("Snippet.ContentHash = %q, want sha256 hash", snippets[0].ContentHash)
	}
}

func TestBuildReviewExternalDocSnippetsFocusesAroundMultipleTerms(t *testing.T) {
	content := "front-marker " +
		strings.Repeat("prefix filler ", 180) +
		"The Responses API previous_response_id value links follow-up requests. " +
		strings.Repeat("middle filler ", 180) +
		"The tool_choice option controls tool selection."

	snippets := buildReviewExternalDocSnippets("external-doc-1", content, false, []ReviewExternalDocFocusTerm{
		{Term: "previous_response_id", Reason: "query focus"},
		{Term: "tool_choice", Reason: "search result title"},
	})

	if len(snippets) != 2 {
		t.Fatalf("Snippets = %d, want two focused snippets", len(snippets))
	}
	snippet := snippets[0]
	if !strings.Contains(snippet.Content, "previous_response_id") {
		t.Fatalf("focused snippet = %q, want focus term", snippet.Content)
	}
	if strings.Contains(snippet.Content, "front-marker") {
		t.Fatalf("focused snippet kept prefix filler instead of focus area: %q", snippet.Content)
	}
	if snippet.FocusTerm != "previous_response_id" || snippet.FocusReason != "query focus" {
		t.Fatalf("focus metadata = (%q, %q), want previous_response_id/query focus", snippet.FocusTerm, snippet.FocusReason)
	}
	if snippets[0].SnippetID != "external-doc-1-snippet-1" || snippets[1].SnippetID != "external-doc-1-snippet-2" {
		t.Fatalf("snippet IDs = %q/%q, want stable sequential IDs", snippets[0].SnippetID, snippets[1].SnippetID)
	}
	if !strings.Contains(snippets[1].Content, "tool_choice") {
		t.Fatalf("second focused snippet = %q, want second focus term", snippets[1].Content)
	}
	if len(snippet.Content) > reviewExternalDocMaxSnippetBytes {
		t.Fatalf("snippet bytes = %d, want <= %d", len(snippet.Content), reviewExternalDocMaxSnippetBytes)
	}
}

func TestBuildReviewExternalDocSnippetsDedupesSameRangeFocuses(t *testing.T) {
	content := "The Responses API previous_response_id field controls follow-up requests and response_format output."

	snippets := buildReviewExternalDocSnippets("external-doc-1", content, false, []ReviewExternalDocFocusTerm{
		{Term: "previous_response_id", Reason: "query focus"},
		{Term: "response_format", Reason: "generic impact token"},
		{Term: "Responses API", Reason: "query subject"},
	})

	if len(snippets) != 1 {
		t.Fatalf("Snippets = %d, want one deduped focused snippet: %#v", len(snippets), snippets)
	}
	if snippets[0].SnippetID != "external-doc-1-snippet-1" {
		t.Fatalf("SnippetID = %q, want external-doc-1-snippet-1", snippets[0].SnippetID)
	}
	if snippets[0].FocusTerm != "previous_response_id" {
		t.Fatalf("FocusTerm = %q, want first matching focus term", snippets[0].FocusTerm)
	}
}

func TestBuildReviewExternalDocSnippetsDedupesHashEquivalentFocuses(t *testing.T) {
	buf := []byte(" " + strings.Repeat("x", reviewExternalDocMaxSnippetBytes-1) + " ")
	buf[599] = 'A'
	buf[600] = 'B'

	snippets := buildReviewExternalDocSnippets("external-doc-1", string(buf), false, []ReviewExternalDocFocusTerm{
		{Term: "A", Reason: "query focus"},
		{Term: "B", Reason: "search result title"},
	})

	if len(snippets) != 1 {
		t.Fatalf("Snippets = %d, want one content-hash-deduped focused snippet: %#v", len(snippets), snippets)
	}
	if snippets[0].FocusTerm != "A" {
		t.Fatalf("FocusTerm = %q, want first hash-equivalent focus term", snippets[0].FocusTerm)
	}
}

func TestBuildReviewExternalDocSnippetsUsesContractFocusBeforeGenericSubject(t *testing.T) {
	candidate := reviewWebSearchEvidenceQueryCandidate{
		query:   "OpenAI API previous_response_id official documentation",
		subject: "OpenAI API",
		focus:   "previous_response_id",
	}
	content := "generic-header OpenAI API overview " +
		strings.Repeat("intro filler ", 180) +
		"The previous_response_id section defines the follow-up request contract."

	snippets := buildReviewExternalDocSnippets(
		"external-doc-1",
		content,
		false,
		buildReviewExternalDocFocusTerms(candidate, ReviewWebSearchEvidenceResult{}, nil),
	)

	if len(snippets) == 0 {
		t.Fatal("Snippets empty, want focused snippet")
	}
	snippet := snippets[0]
	if !strings.Contains(snippet.Content, "previous_response_id") {
		t.Fatalf("focused snippet = %q, want contract focus term", snippet.Content)
	}
	if strings.Contains(snippet.Content, "generic-header") {
		t.Fatalf("focused snippet used generic subject prefix instead of contract focus: %q", snippet.Content)
	}
	if snippet.FocusTerm != "previous_response_id" || snippet.FocusReason != "query focus" {
		t.Fatalf("focus metadata = (%q, %q), want previous_response_id/query focus", snippet.FocusTerm, snippet.FocusReason)
	}
}

func TestBuildReviewExternalDocSnippetsFallsBackToPrefixWhenFocusTermMissing(t *testing.T) {
	content := "front-marker " + strings.Repeat("prefix filler ", 120) + "later target text"

	snippets := buildReviewExternalDocSnippets("external-doc-1", content, false, []ReviewExternalDocFocusTerm{
		{Term: "missing_focus_token", Reason: "query focus"},
	})

	if len(snippets) == 0 {
		t.Fatal("Snippets empty, want prefix fallback")
	}
	if !strings.HasPrefix(snippets[0].Content, "front-marker") {
		t.Fatalf("fallback snippet = %q, want prefix chunk", snippets[0].Content)
	}
	if snippets[0].FocusTerm != "" || snippets[0].FocusReason != "" {
		t.Fatalf("fallback focus metadata = (%q, %q), want empty", snippets[0].FocusTerm, snippets[0].FocusReason)
	}
}

func TestBuildReviewExternalDocSnippetsHashesStableFocusedContent(t *testing.T) {
	content := strings.Repeat("intro ", 180) + "FocusAnchor stable text" + strings.Repeat(" outro", 180)
	focusTerms := []ReviewExternalDocFocusTerm{{Term: "FocusAnchor", Reason: "query focus"}}

	first := buildReviewExternalDocSnippets("external-doc-1", content, false, focusTerms)
	second := buildReviewExternalDocSnippets("external-doc-1", content, false, focusTerms)

	if len(first) != 1 || len(second) != 1 {
		t.Fatalf("snippet counts = (%d, %d), want one each", len(first), len(second))
	}
	if first[0].Content != second[0].Content {
		t.Fatalf("snippet content changed:\nfirst=%q\nsecond=%q", first[0].Content, second[0].Content)
	}
	if first[0].ContentHash != reviewExternalDocContentHash(first[0].Content) {
		t.Fatalf("Snippet.ContentHash = %q, want hash of snippet content", first[0].ContentHash)
	}
	if first[0].ContentHash != second[0].ContentHash {
		t.Fatalf("snippet hash changed: %q vs %q", first[0].ContentHash, second[0].ContentHash)
	}
}
