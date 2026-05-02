package review

import (
	"context"
	"errors"
	"testing"
)

func TestProbeRunner_UnsupportedMode(t *testing.T) {
	repo := newProbeTestRepo(t)
	runner := NewProbeRunner(repo)

	result, err := runner.Run(context.Background(), ReviewProbeRequest{
		ID:   "probe-unsupported",
		Mode: ReviewProbeScratchOnly,
	})
	if err == nil {
		t.Fatal("Run() error = nil, want unsupported mode error")
	}
	if !errors.Is(err, ErrUnsupportedReviewProbeMode) {
		t.Fatalf("Run() error = %v, want ErrUnsupportedReviewProbeMode", err)
	}
	if result.Status != ReviewProbeBlocked {
		t.Fatalf("Status = %q, want %q", result.Status, ReviewProbeBlocked)
	}
}
