package review

import (
	"reflect"
	"testing"
)

func TestCanonicalizeReviewProbeResultMutationOutcome(t *testing.T) {
	tests := []struct {
		name string
		in   ReviewProbeResult
		want ReviewProbeResult
	}{
		{
			name: "status only mutation canonicalizes flag",
			in: ReviewProbeResult{
				ID:     "probe-1",
				Mode:   ReviewProbeHostReadOnly,
				Status: ReviewProbeMutatedWorktree,
			},
			want: ReviewProbeResult{
				ID:              "probe-1",
				Mode:            ReviewProbeHostReadOnly,
				Status:          ReviewProbeMutatedWorktree,
				MutatedWorktree: true,
			},
		},
		{
			name: "flag only mutation canonicalizes status",
			in: ReviewProbeResult{
				ID:              "probe-1",
				Mode:            ReviewProbeHostReadOnly,
				Status:          ReviewProbeFailed,
				MutatedWorktree: true,
			},
			want: ReviewProbeResult{
				ID:              "probe-1",
				Mode:            ReviewProbeHostReadOnly,
				Status:          ReviewProbeMutatedWorktree,
				MutatedWorktree: true,
			},
		},
		{
			name: "non mutation is unchanged",
			in: ReviewProbeResult{
				ID:           "probe-1",
				Mode:         ReviewProbeHostReadOnly,
				Status:       ReviewProbeFailed,
				MutatedFiles: []string{"internal/review/runner.go"},
				Error:        "exit status 1",
			},
			want: ReviewProbeResult{
				ID:           "probe-1",
				Mode:         ReviewProbeHostReadOnly,
				Status:       ReviewProbeFailed,
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
		in   ReviewProbeSummary
		want ReviewProbeSummary
	}{
		{
			name: "status only mutation canonicalizes flag",
			in: ReviewProbeSummary{
				ProbeID: "probe-1",
				Mode:    ReviewProbeHostReadOnly,
				Status:  ReviewProbeMutatedWorktree,
			},
			want: ReviewProbeSummary{
				ProbeID:         "probe-1",
				Mode:            ReviewProbeHostReadOnly,
				Status:          ReviewProbeMutatedWorktree,
				MutatedWorktree: true,
			},
		},
		{
			name: "flag only mutation canonicalizes status",
			in: ReviewProbeSummary{
				ProbeID:         "probe-1",
				Mode:            ReviewProbeHostReadOnly,
				Status:          ReviewProbeFailed,
				MutatedWorktree: true,
			},
			want: ReviewProbeSummary{
				ProbeID:         "probe-1",
				Mode:            ReviewProbeHostReadOnly,
				Status:          ReviewProbeMutatedWorktree,
				MutatedWorktree: true,
			},
		},
		{
			name: "non mutation is unchanged",
			in: ReviewProbeSummary{
				ProbeID:      "probe-1",
				Mode:         ReviewProbeHostReadOnly,
				Status:       ReviewProbeFailed,
				MutatedFiles: []string{"internal/review/runner.go"},
				Error:        "exit status 1",
			},
			want: ReviewProbeSummary{
				ProbeID:      "probe-1",
				Mode:         ReviewProbeHostReadOnly,
				Status:       ReviewProbeFailed,
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
