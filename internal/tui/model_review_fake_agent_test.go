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
	err                       error
	statusLineAfterReview     string
	statusSnapshotAfterReview StatusSnapshot
}

func (s *reviewCapableStubAgent) RunReview(_ context.Context, req review.ReviewRequest) (review.ReviewReport, error) {
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
	return s.report, s.err
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

func (s *cancellableReviewAgent) RunReview(ctx context.Context, _ review.ReviewRequest) (review.ReviewReport, error) {
	close(s.started)
	<-ctx.Done()
	return review.ReviewReport{}, ctx.Err()
}
