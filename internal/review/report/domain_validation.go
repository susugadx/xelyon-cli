package report

import "github.com/susugadx/xelyon-cli/internal/review/domain"

func isKnownReviewProbeMode(mode ReviewProbeMode) bool {
	return domain.IsKnownReviewProbeMode(mode)
}

func isKnownReviewProbeStatus(status ReviewProbeStatus) bool {
	return domain.IsKnownReviewProbeStatus(status)
}
