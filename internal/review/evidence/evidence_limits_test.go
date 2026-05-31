package evidence

import (
	"testing"
	"time"
)

func TestDefaultReviewEvidenceLimits(t *testing.T) {
	got := DefaultReviewEvidenceLimits()
	want := ReviewEvidenceLimits{
		MaxCommandOutputBytes:      1024 * 1024,
		MaxUntrackedFileBytes:      64 * 1024,
		MaxRuleFileBytes:           64 * 1024,
		MaxTotalUntrackedBytes:     256 * 1024,
		MaxUntrackedFiles:          100,
		MaxContextFileBytes:        32 * 1024,
		MaxTotalContextBytes:       160 * 1024,
		MaxContextFiles:            24,
		MaxRelatedSearchTerms:      12,
		MaxRelatedSearchFiles:      200,
		MaxTotalRelatedSearchBytes: 512 * 1024,
		MaxRelatedSearchFileBytes:  64 * 1024,
		MaxRelatedSearchHits:       40,
		MaxSearchSnippetBytes:      240,
		CommandTimeout:             30 * time.Second,
	}
	if got != want {
		t.Fatalf("DefaultReviewEvidenceLimits() = %#v, want %#v", got, want)
	}
}
