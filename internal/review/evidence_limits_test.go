package review

import (
	"testing"
	"time"
)

func TestDefaultReviewEvidenceLimits(t *testing.T) {
	got := DefaultReviewEvidenceLimits()
	want := ReviewEvidenceLimits{
		MaxCommandOutputBytes:  1024 * 1024,
		MaxUntrackedFileBytes:  64 * 1024,
		MaxRuleFileBytes:       64 * 1024,
		MaxTotalUntrackedBytes: 256 * 1024,
		MaxUntrackedFiles:      100,
		CommandTimeout:         30 * time.Second,
	}
	if got != want {
		t.Fatalf("DefaultReviewEvidenceLimits() = %#v, want %#v", got, want)
	}
}
