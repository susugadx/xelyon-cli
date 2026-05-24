package tui

import (
	"context"

	"github.com/susugadx/xelyon-cli/internal/review"
)

type reviewCapableStubAgent struct {
	stubAgent
	reviewCalls               int
	lastRequest               review.ReviewRequest
	report                    review.ReviewReport
	usage                     ReviewRunUsageSummary
	err                       error
	statusLineAfterReview     string
	statusSnapshotAfterReview StatusSnapshot
}

func (s *reviewCapableStubAgent) RunReview(_ context.Context, req review.ReviewRequest) (ReviewRunResult, error) {
	s.reviewCalls++
	s.lastRequest = req
	if s.statusLineAfterReview != "" || s.statusSnapshotAfterReview != (StatusSnapshot{}) {
		s.mu.Lock()
		if s.statusLineAfterReview != "" {
			s.statusLine = s.statusLineAfterReview
		}
		if s.statusSnapshotAfterReview != (StatusSnapshot{}) {
			s.statusSnapshot = s.statusSnapshotAfterReview
		}
		s.mu.Unlock()
	}
	return ReviewRunResult{Report: s.report, Usage: s.usage}, s.err
}

type cancellableReviewAgent struct {
	stubAgent
	started chan struct{}
}

func newCancellableReviewAgent() *cancellableReviewAgent {
	return &cancellableReviewAgent{
		stubAgent: stubAgent{statusLine: "ready"},
		started:   make(chan struct{}),
	}
}

func (s *cancellableReviewAgent) RunReview(ctx context.Context, _ review.ReviewRequest) (ReviewRunResult, error) {
	close(s.started)
	<-ctx.Done()
	return ReviewRunResult{}, ctx.Err()
}
