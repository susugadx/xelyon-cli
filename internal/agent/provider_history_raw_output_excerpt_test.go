package agent

import (
	"strconv"
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/token"
)

func TestProviderHistoryRawOutputBodyCoverageExcerptShrinksPreRenderedMatchedExcerpt(t *testing.T) {
	lines := []string{
		"before line 1 " + strings.Repeat("context ", 80),
		"before line 2 " + strings.Repeat("context ", 80),
		"before line 3 " + strings.Repeat("context ", 80),
		"target-5555 stable matched payload",
		"after line 1 " + strings.Repeat("context ", 80),
		"after line 2 " + strings.Repeat("context ", 80),
		"after line 3 " + strings.Repeat("context ", 80),
	}
	full := providerHistoryRawOutputRenderMatchedExcerpt(lines, "target-5555", 3, 10, 0, len(lines))
	matchOnly := providerHistoryRawOutputRenderMatchedExcerpt([]string{lines[3]}, "target-5555", 3, 10, 3, 4)
	budget := token.EstimateTokenCount(matchOnly) + 4
	if token.EstimateTokenCount(full) <= budget {
		t.Fatalf("test setup invalid: full excerpt already fits budget %d", budget)
	}

	excerpt, reason := providerHistoryRawOutputBodyCoverageExcerpt(full, budget, []string{"target-5555"})
	if reason != "" {
		t.Fatalf("BodyCoverageExcerpt() reason = %q, want shrink success", reason)
	}
	if token.EstimateTokenCount(excerpt) > budget {
		t.Fatalf("shrunk excerpt tokens = %d, want <= %d:\n%s", token.EstimateTokenCount(excerpt), budget, excerpt)
	}
	if !strings.Contains(excerpt, "target-5555 stable matched payload") {
		t.Fatalf("shrunk excerpt missing matched payload:\n%s", excerpt)
	}
	if strings.Contains(excerpt, "before line 1") || strings.Contains(excerpt, "after line 3") {
		t.Fatalf("shrunk excerpt retained far context lines:\n%s", excerpt)
	}
}

func TestProviderHistoryRawOutputBodyCoverageExcerptKeepsFailClosedWhenMatchedExcerptStillTooLarge(t *testing.T) {
	lines := []string{
		"before line",
		"target-7777 stable matched payload",
		"after line",
	}
	full := providerHistoryRawOutputRenderMatchedExcerpt(lines, "target-7777", 1, 3, 0, len(lines))

	excerpt, reason := providerHistoryRawOutputBodyCoverageExcerpt(full, 1, []string{"target-7777"})
	if excerpt != "" || reason != providerHistoryRawOutputActiveContextCoverageInsufficientReason {
		t.Fatalf("BodyCoverageExcerpt() = (%q, %q), want coverage insufficient", excerpt, reason)
	}
}

func TestProviderHistoryRawOutputBodyCoverageExcerptPreservesQuotedMatchedTerm(t *testing.T) {
	term := `target; "quoted" payload`
	lines := []string{
		"before line " + strings.Repeat("context ", 80),
		term + " stable matched payload",
		"after line " + strings.Repeat("context ", 80),
	}
	full := providerHistoryRawOutputRenderMatchedExcerpt(lines, term, 1, 3, 0, len(lines))
	matchOnly := providerHistoryRawOutputRenderMatchedExcerpt([]string{lines[1]}, term, 1, 3, 1, 2)
	budget := token.EstimateTokenCount(matchOnly) + 4
	if token.EstimateTokenCount(full) <= budget {
		t.Fatalf("test setup invalid: full excerpt already fits budget %d", budget)
	}

	excerpt, reason := providerHistoryRawOutputBodyCoverageExcerpt(full, budget, []string{term})
	if reason != "" {
		t.Fatalf("BodyCoverageExcerpt() reason = %q, want shrink success", reason)
	}
	if !strings.Contains(excerpt, "matched_term="+strconv.Quote(term)+";") {
		t.Fatalf("shrunk excerpt lost quoted matched term:\n%s", excerpt)
	}
}
