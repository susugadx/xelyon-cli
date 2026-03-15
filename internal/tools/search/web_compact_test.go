package search

import (
	"fmt"
	"strings"
	"testing"
)

func TestCompactWebSearchResult_ShortResult(t *testing.T) {
	short := "Summary:\nGo 1.24 was released.\n\nSources:\n\n1. Go Blog\n   URL: https://go.dev/blog"
	got := CompactWebSearchResult("Go 1.24 release", short)
	// short result should preserve content and prepend query
	if !strings.Contains(got, "Query: Go 1.24 release") {
		t.Error("should contain Query: header")
	}
	if !strings.Contains(got, "Go 1.24 was released") {
		t.Error("should preserve original summary")
	}
}

func TestCompactWebSearchResult_NoResults(t *testing.T) {
	got := CompactWebSearchResult("test query", "No results found.")
	if got != "No results found." {
		t.Errorf("'No results found.' should be unchanged, got: %s", got)
	}
}

func TestCompactWebSearchResult_Empty(t *testing.T) {
	got := CompactWebSearchResult("test query", "")
	if got != "" {
		t.Errorf("empty should be unchanged, got: %s", got)
	}
}

func TestCompactWebSearchResult_LongSummaryTruncated(t *testing.T) {
	var lines []string
	lines = append(lines, "Summary:")
	for i := 0; i < 30; i++ {
		lines = append(lines, fmt.Sprintf("This is paragraph %d of the summary with detailed information.", i))
	}
	lines = append(lines, "")
	lines = append(lines, "Sources:")
	lines = append(lines, "")
	lines = append(lines, "1. Example")
	lines = append(lines, "   URL: https://example.com")

	result := strings.Join(lines, "\n")
	got := CompactWebSearchResult("detailed topic", result)

	if !strings.Contains(got, "Query: detailed topic") {
		t.Error("should contain Query: header")
	}
	if !strings.Contains(got, "Summary:") {
		t.Error("should contain Summary: header")
	}
	if !strings.Contains(got, "summary truncated") {
		t.Error("should indicate summary was truncated")
	}
	if !strings.Contains(got, "https://example.com") {
		t.Error("should preserve source URL")
	}
}

func TestCompactWebSearchResult_SourcesCompacted(t *testing.T) {
	var b strings.Builder
	b.WriteString("Summary:\nBrief finding.\n\nSources:\n")
	for i := 1; i <= 12; i++ {
		fmt.Fprintf(&b, "\n%d. Source %d Title\n   URL: https://example.com/%d\n", i, i, i)
	}

	result := b.String()
	got := CompactWebSearchResult("test sources", result)

	// should have compact format
	if !strings.Contains(got, "Sources:") {
		t.Error("should contain Sources: header")
	}

	// first 7 sources should be preserved
	for i := 1; i <= 7; i++ {
		url := fmt.Sprintf("https://example.com/%d", i)
		if !strings.Contains(got, url) {
			t.Errorf("should contain source URL %s", url)
		}
	}

	// source 8+ should be omitted
	if strings.Contains(got, "https://example.com/8") {
		t.Error("source 8 should be omitted (max 7)")
	}

	// should indicate exactly 5 more sources (12 - 7 = 5)
	if !strings.Contains(got, "[+5 more sources]") {
		t.Errorf("should show [+5 more sources], got:\n%s", got)
	}

	// compact single-line format
	if !strings.Contains(got, "- Source 1 Title (https://example.com/1)") {
		t.Errorf("should use compact single-line format, got:\n%s", got)
	}
}

func TestCompactWebSearchResult_SourcesCompacted_ExactCount(t *testing.T) {
	// 10 sources, max 7 → exactly 3 omitted
	var b strings.Builder
	b.WriteString("Summary:\nFindings about the topic with enough content to exceed the minimum length threshold.\n")
	b.WriteString(strings.Repeat("Additional context. ", 20))
	b.WriteString("\n\nSources:\n")
	for i := 1; i <= 10; i++ {
		fmt.Fprintf(&b, "\n%d. Title %d\n   URL: https://example.com/%d\n", i, i, i)
	}

	got := CompactWebSearchResult("count test", b.String())

	if !strings.Contains(got, "[+3 more sources]") {
		t.Errorf("10 sources with max 7 should show [+3 more sources], got:\n%s", got)
	}
}

func TestCompactWebSearchResult_QueryPreserved(t *testing.T) {
	var b strings.Builder
	b.WriteString("Summary:\nResults about Go generics implementation.\n")
	for i := 0; i < 20; i++ {
		fmt.Fprintf(&b, "Detail line %d about generics implementation.\n", i)
	}
	b.WriteString("\nSources:\n\n1. Go Blog\n   URL: https://go.dev/blog/generics\n")

	query := "Go generics type constraints"
	got := CompactWebSearchResult(query, b.String())

	// query must appear as explicit Query: header, not just in summary text
	if !strings.HasPrefix(got, "Query: Go generics type constraints\n") {
		t.Errorf("should start with Query: header, got:\n%s", got[:80])
	}
	if !strings.Contains(got, "https://go.dev/blog/generics") {
		t.Error("source URL should be preserved")
	}
}

func TestCompactWebSearchResult_QueryPreserved_Short(t *testing.T) {
	// even short results should have Query: header
	short := "Summary:\nBrief result."
	got := CompactWebSearchResult("my search", short)
	if !strings.HasPrefix(got, "Query: my search\n") {
		t.Errorf("short result should have Query: header, got:\n%s", got)
	}
}

func TestCompactWebSearchResult_FindingsPreserved(t *testing.T) {
	var b strings.Builder
	b.WriteString("Summary:\nKey finding: React 19 introduces server components.\n")
	for i := 0; i < 20; i++ {
		fmt.Fprintf(&b, "Additional detail line %d.\n", i)
	}
	b.WriteString("\nSources:\n\n1. React Blog\n   URL: https://react.dev/blog\n")

	got := CompactWebSearchResult("React 19 features", b.String())

	if !strings.Contains(got, "server components") {
		t.Error("key findings should be preserved in first lines of summary")
	}
}

func TestCompactWebSearchResult_SourceURLsNotDropped(t *testing.T) {
	var b strings.Builder
	b.WriteString("Summary:\nShort summary.\n")
	b.WriteString(strings.Repeat("padding text. ", 50))
	b.WriteString("\n\nSources:\n")
	urls := []string{
		"https://docs.python.org/3/",
		"https://pypi.org/project/requests/",
		"https://flask.palletsprojects.com/",
	}
	for i, url := range urls {
		fmt.Fprintf(&b, "\n%d. Source %d\n   URL: %s\n", i+1, i+1, url)
	}

	got := CompactWebSearchResult("python libraries", b.String())

	for _, url := range urls {
		if !strings.Contains(got, url) {
			t.Errorf("source URL %s should be preserved", url)
		}
	}
}

// ── splitWebSearchSections ──

func TestSplitWebSearchSections(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantSumm string
		wantSrc  string
	}{
		{
			name:     "standard format",
			input:    "Summary:\nSome findings.\n\nSources:\n\n1. Title\n   URL: https://example.com",
			wantSumm: "Summary:\nSome findings.",
			wantSrc:  "Sources:\n\n1. Title\n   URL: https://example.com",
		},
		{
			name:     "no sources",
			input:    "Summary:\nOnly summary here.",
			wantSumm: "Summary:\nOnly summary here.",
			wantSrc:  "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotSumm, gotSrc := splitWebSearchSections(tt.input)
			if gotSumm != tt.wantSumm {
				t.Errorf("summary = %q, want %q", gotSumm, tt.wantSumm)
			}
			if gotSrc != tt.wantSrc {
				t.Errorf("sources = %q, want %q", gotSrc, tt.wantSrc)
			}
		})
	}
}

// ── countRemainingEntries ──

func TestCountRemainingEntries(t *testing.T) {
	lines := []string{
		"8. Title 8",
		"   URL: https://example.com/8",
		"",
		"9. Title 9",
		"   URL: https://example.com/9",
	}
	got := countRemainingEntries(lines)
	if got != 2 {
		t.Errorf("countRemainingEntries = %d, want 2", got)
	}
}
