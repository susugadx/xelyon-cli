package review

import (
	"reflect"
	"testing"

	reviewprobe "github.com/susugadx/xelyon-cli/internal/review/probe"
	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
)

func TestCanonicalizeReviewProbeResultMutationOutcome(t *testing.T) {
	tests := []struct {
		name string
		in   reviewprobe.ReviewProbeResult
		want reviewprobe.ReviewProbeResult
	}{
		{
			name: "status only mutation canonicalizes flag",
			in: reviewprobe.ReviewProbeResult{
				ID:     "probe-1",
				Mode:   reviewprobe.ReviewProbeHostReadOnly,
				Status: reviewprobe.ReviewProbeMutatedWorktree,
			},
			want: reviewprobe.ReviewProbeResult{
				ID:              "probe-1",
				Mode:            reviewprobe.ReviewProbeHostReadOnly,
				Status:          reviewprobe.ReviewProbeMutatedWorktree,
				MutatedWorktree: true,
			},
		},
		{
			name: "flag only mutation canonicalizes status",
			in: reviewprobe.ReviewProbeResult{
				ID:              "probe-1",
				Mode:            reviewprobe.ReviewProbeHostReadOnly,
				Status:          reviewprobe.ReviewProbeFailed,
				MutatedWorktree: true,
			},
			want: reviewprobe.ReviewProbeResult{
				ID:              "probe-1",
				Mode:            reviewprobe.ReviewProbeHostReadOnly,
				Status:          reviewprobe.ReviewProbeMutatedWorktree,
				MutatedWorktree: true,
			},
		},
		{
			name: "non mutation is unchanged",
			in: reviewprobe.ReviewProbeResult{
				ID:           "probe-1",
				Mode:         reviewprobe.ReviewProbeHostReadOnly,
				Status:       reviewprobe.ReviewProbeFailed,
				MutatedFiles: []string{"internal/review/runner.go"},
				Error:        "exit status 1",
			},
			want: reviewprobe.ReviewProbeResult{
				ID:           "probe-1",
				Mode:         reviewprobe.ReviewProbeHostReadOnly,
				Status:       reviewprobe.ReviewProbeFailed,
				MutatedFiles: []string{"internal/review/runner.go"},
				Error:        "exit status 1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := canonicalizeReviewProbeResultMutationOutcome(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("canonicalizeReviewProbeResultMutationOutcome() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestCanonicalizeReviewProbeSummaryMutationOutcome(t *testing.T) {
	tests := []struct {
		name string
		in   reviewreport.ReviewProbeSummary
		want reviewreport.ReviewProbeSummary
	}{
		{
			name: "status only mutation canonicalizes flag",
			in: reviewreport.ReviewProbeSummary{
				ProbeID: "probe-1",
				Mode:    reviewprobe.ReviewProbeHostReadOnly,
				Status:  reviewprobe.ReviewProbeMutatedWorktree,
			},
			want: reviewreport.ReviewProbeSummary{
				ProbeID:         "probe-1",
				Mode:            reviewprobe.ReviewProbeHostReadOnly,
				Status:          reviewprobe.ReviewProbeMutatedWorktree,
				MutatedWorktree: true,
			},
		},
		{
			name: "flag only mutation canonicalizes status",
			in: reviewreport.ReviewProbeSummary{
				ProbeID:         "probe-1",
				Mode:            reviewprobe.ReviewProbeHostReadOnly,
				Status:          reviewprobe.ReviewProbeFailed,
				MutatedWorktree: true,
			},
			want: reviewreport.ReviewProbeSummary{
				ProbeID:         "probe-1",
				Mode:            reviewprobe.ReviewProbeHostReadOnly,
				Status:          reviewprobe.ReviewProbeMutatedWorktree,
				MutatedWorktree: true,
			},
		},
		{
			name: "non mutation is unchanged",
			in: reviewreport.ReviewProbeSummary{
				ProbeID:      "probe-1",
				Mode:         reviewprobe.ReviewProbeHostReadOnly,
				Status:       reviewprobe.ReviewProbeFailed,
				MutatedFiles: []string{"internal/review/runner.go"},
				Error:        "exit status 1",
			},
			want: reviewreport.ReviewProbeSummary{
				ProbeID:      "probe-1",
				Mode:         reviewprobe.ReviewProbeHostReadOnly,
				Status:       reviewprobe.ReviewProbeFailed,
				MutatedFiles: []string{"internal/review/runner.go"},
				Error:        "exit status 1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := canonicalizeReviewProbeSummaryMutationOutcome(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("canonicalizeReviewProbeSummaryMutationOutcome() = %#v, want %#v", got, tt.want)
			}
		})
	}
}
