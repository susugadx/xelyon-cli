package tui

import (
	"time"

	reviewdomain "github.com/susugadx/xelyon-cli/internal/review/domain"
	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
)

func newTUITestReviewReport() reviewreport.ReviewReport {
	return reviewreport.ReviewReport{
		SchemaVersion:             reviewreport.ReviewReportSchemaVersionV2,
		TargetKind:                reviewdomain.TargetCurrentChanges,
		GeneratedAt:               time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
		OverallVerificationStatus: reviewreport.ReviewVerificationVerified,
		Verdict:                   reviewreport.ReviewVerdictHasFindings,
		Summary:                   "Review found one issue.",
		RootCauseGroups: []reviewreport.ReviewRootCauseGroup{{
			ID:                 "request-state",
			Title:              "request state",
			Severity:           reviewreport.ReviewGroupSeverityMedium,
			VerificationStatus: reviewreport.ReviewVerificationVerified,
			Findings: []reviewreport.ReviewFinding{{
				ID:    "stale-result",
				Title: "stale result is ignored",
			}},
		}},
	}
}
