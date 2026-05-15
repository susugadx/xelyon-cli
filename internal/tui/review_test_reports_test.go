package tui

import (
	"fmt"
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

func newLongTUITestReviewReport(groupCount int) review.ReviewReport {
	report := newTUITestReviewReport()
	report.Summary = "Review found multiple issues."
	report.RootCauseGroups = make([]review.ReviewRootCauseGroup, 0, groupCount)
	for i := 0; i < groupCount; i++ {
		report.RootCauseGroups = append(report.RootCauseGroups, review.ReviewRootCauseGroup{
			ID:                 fmt.Sprintf("hidden-group-%02d", i),
			Title:              fmt.Sprintf("hidden group %02d", i),
			Severity:           review.ReviewGroupSeverityMedium,
			VerificationStatus: review.ReviewVerificationVerified,
			Findings: []review.ReviewFinding{{
				ID:    fmt.Sprintf("hidden-finding-%02d", i),
				Title: fmt.Sprintf("hidden finding %02d", i),
			}},
		})
	}
	return report
}
