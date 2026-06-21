package probeplan

import (
	"github.com/susugadx/xelyon-cli/internal/review/domain"
	"github.com/susugadx/xelyon-cli/internal/review/report"
)

func isKnownReviewProbeMode(mode domain.ReviewProbeMode) bool {
	return domain.IsKnownReviewProbeMode(mode)
}

func validateEvidenceRef(field string, ref report.ReviewEvidenceRef, probeSummariesByID map[string]report.ReviewProbeSummary) error {
	return report.ValidateEvidenceRef(field, ref, probeSummariesByID)
}

func isKnownReviewGroupSeverity(severity report.ReviewGroupSeverity) bool {
	return report.IsKnownReviewGroupSeverity(severity)
}

func isKnownReviewEvidenceKind(kind string) bool {
	return report.IsKnownReviewEvidenceKind(kind)
}
