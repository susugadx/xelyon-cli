package report

import "github.com/susugadx/xelyon-cli/internal/review/domain"

func isKnownReviewProbeMode(mode domain.ReviewProbeMode) bool {
	return domain.IsKnownReviewProbeMode(mode)
}

func isKnownReviewProbeStatus(status domain.ReviewProbeStatus) bool {
	return domain.IsKnownReviewProbeStatus(status)
}
