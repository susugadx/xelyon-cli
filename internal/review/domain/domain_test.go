package domain

import "testing"

func TestIsKnownReviewProbeMode(t *testing.T) {
	tests := []struct {
		name string
		mode ReviewProbeMode
		want bool
	}{
		{name: "host read only", mode: ReviewProbeHostReadOnly, want: true},
		{name: "scratch only", mode: ReviewProbeScratchOnly, want: true},
		{name: "repo sandbox", mode: ReviewProbeRepoSandbox, want: true},
		{name: "empty", mode: "", want: false},
		{name: "unknown", mode: ReviewProbeMode("unknown"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsKnownReviewProbeMode(tt.mode); got != tt.want {
				t.Fatalf("IsKnownReviewProbeMode(%q) = %v, want %v", tt.mode, got, tt.want)
			}
		})
	}
}

func TestIsKnownReviewProbeStatus(t *testing.T) {
	tests := []struct {
		name   string
		status ReviewProbeStatus
		want   bool
	}{
		{name: "passed", status: ReviewProbePassed, want: true},
		{name: "failed", status: ReviewProbeFailed, want: true},
		{name: "blocked", status: ReviewProbeBlocked, want: true},
		{name: "timed out", status: ReviewProbeTimedOut, want: true},
		{name: "mutated worktree", status: ReviewProbeMutatedWorktree, want: true},
		{name: "empty", status: "", want: false},
		{name: "unknown", status: ReviewProbeStatus("unknown"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsKnownReviewProbeStatus(tt.status); got != tt.want {
				t.Fatalf("IsKnownReviewProbeStatus(%q) = %v, want %v", tt.status, got, tt.want)
			}
		})
	}
}
