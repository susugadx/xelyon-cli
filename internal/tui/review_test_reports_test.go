package tui

import (
	"time"

	"github.com/susugadx/xelyon-cli/internal/review"
)

func newTUITestReviewReport() review.ReviewReport {
	return review.ReviewReport{
		SchemaVersion:             review.ReviewReportSchemaVersionV2,
		TargetKind:                review.TargetCurrentChanges,
		GeneratedAt:               time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
		OverallVerificationStatus: review.ReviewVerificationVerified,
		Verdict:                   review.ReviewVerdictHasFindings,
		Summary:                   "Review found one issue.",
		RootCauseGroups: []review.ReviewRootCauseGroup{{
			ID:                 "request-state",
			Title:              "request state",
			Severity:           review.ReviewGroupSeverityMedium,
			VerificationStatus: review.ReviewVerificationVerified,
			Findings: []review.ReviewFinding{{
				ID:    "stale-result",
				Title: "stale result is ignored",
			}},
		}},
	}
}
