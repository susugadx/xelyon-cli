package agent

import (
	"fmt"
	"strings"
	"testing"
)

func TestCompactWebSearch_ShortResult(t *testing.T) {
	short := "Summary:\nBrief finding.\n\nSources:\n- Title (https://example.com)"
	got := compactWebSearch(short)
	if got != short {
		t.Errorf("short result should be unchanged, got:\n%s", got)
	}
}

func TestCompactWebSearch_LongSummaryTruncated(t *testing.T) {
	var b strings.Builder
	b.WriteString("Summary:\n")
	for i := 0; i < 20; i++ {
		fmt.Fprintf(&b, "Detail line %d with lots of detailed information about the topic at hand.\n", i)
	}
	b.WriteString("\nSources:\n- Title (https://example.com)")

	result := b.String()
	got := compactWebSearch(result)

	if !strings.Contains(got, "Summary:") {
		t.Error("should preserve Summary: header")
	}
	if !strings.Contains(got, "truncated for history") {
		t.Error("should indicate truncation")
	}
	if !strings.Contains(got, "https://example.com") {
		t.Error("should preserve source URL")
	}
	if len(got) >= len(result) {
		t.Errorf("compacted (%d) should be shorter than original (%d)", len(got), len(result))
	}
}

func TestCompactWebSearch_SourcesCapped(t *testing.T) {
	var b strings.Builder
	b.WriteString("Summary:\nFindings.\n")
	b.WriteString(strings.Repeat("padding. ", 80)) // make it long enough
	b.WriteString("\n\nSources:")
	for i := 1; i <= 10; i++ {
		fmt.Fprintf(&b, "\n- Source %d (https://example.com/%d)", i, i)
	}

	result := b.String()
	got := compactWebSearch(result)

	// first 5 sources should be preserved
	for i := 1; i <= 5; i++ {
		url := fmt.Sprintf("https://example.com/%d", i)
		if !strings.Contains(got, url) {
			t.Errorf("source %d URL should be preserved", i)
		}
	}

	// source 6+ should be omitted
	if strings.Contains(got, "https://example.com/6") {
		t.Error("source 6 should be omitted (max 5 in history)")
	}

	if !strings.Contains(got, "omitted for history") {
		t.Error("should indicate sources were omitted")
	}
}

func TestCompactWebSearch_NoSourcesSection(t *testing.T) {
	var lines []string
	lines = append(lines, "Summary:")
	for i := 0; i < 20; i++ {
		lines = append(lines, fmt.Sprintf("Very long line %d with lots of content padding.", i))
	}
	result := strings.Join(lines, "\n")

	// ensure it's long enough
	if len(result) < compactWebSearchMinLen {
		result += strings.Repeat(" padding", 100)
	}

	got := compactWebSearch(result)

	if !strings.Contains(got, "Summary:") {
		t.Error("should preserve Summary: header")
	}
	if !strings.Contains(got, "truncated for history") {
		t.Error("should truncate long summary-only result")
	}
}

func TestCompactWebSearch_QueryAndFindingsPreserved(t *testing.T) {
	var b strings.Builder
	b.WriteString("Summary:\nQuery about Rust async runtime.\n")
	b.WriteString("Key finding: tokio is the most popular async runtime.\n")
	for i := 0; i < 15; i++ {
		fmt.Fprintf(&b, "Additional detail %d.\n", i)
	}
	b.WriteString("\nSources:\n- Tokio docs (https://tokio.rs)")

	result := b.String()
	if len(result) < compactWebSearchMinLen {
		// pad to exceed threshold
		result = strings.Replace(result, "Additional detail 0.", "Additional detail 0."+strings.Repeat(" padding", 50), 1)
	}
	got := compactWebSearch(result)

	if !strings.Contains(got, "Rust async runtime") {
		t.Error("query context should be preserved")
	}
	if !strings.Contains(got, "tokio") {
		t.Error("key finding should be preserved")
	}
	if !strings.Contains(got, "https://tokio.rs") {
		t.Error("source URL should be preserved")
	}
}
