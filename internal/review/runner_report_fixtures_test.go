package review

import (
	"encoding/json"
	"testing"
	"time"

	reviewdomain "github.com/susugadx/xelyon-cli/internal/review/domain"
	reviewreport "github.com/susugadx/xelyon-cli/internal/review/report"
)

func newFileEvidenceRefForValidationTest() reviewreport.ReviewEvidenceRef {
	return reviewreport.ReviewEvidenceRef{
		Kind: reviewreport.ReviewEvidenceKindFile,
		Path: "internal/review/report_validation.go",
		Line: 1,
	}
}

func newPlanAwareHasFindingsReportForValidationTest() reviewreport.ReviewReport {
	return reviewreport.ReviewReport{
		SchemaVersion:             reviewreport.ReviewReportSchemaVersionV2,
		TargetKind:                reviewdomain.TargetCurrentChanges,
		GeneratedAt:               time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
		OverallVerificationStatus: reviewreport.ReviewVerificationVerified,
		Verdict:                   reviewreport.ReviewVerdictHasFindings,
		RootCauseGroups: []reviewreport.ReviewRootCauseGroup{
			{
				ID:                 "rc-1",
				Title:              "test group",
				Severity:           reviewreport.ReviewGroupSeverityLow,
				VerificationStatus: reviewreport.ReviewVerificationVerified,
				FixStrategy:        "fix root cause",
				VerificationPlan:   []string{"run focused validation"},
				Findings: []reviewreport.ReviewFinding{
					{
						ID:           "finding-1",
						Title:        "finding",
						EvidenceRefs: []reviewreport.ReviewEvidenceRef{newFileEvidenceRefForValidationTest()},
					},
				},
			},
		},
		ScopeCoverage: &reviewreport.ReviewReportScopeCoverage{
			ReviewedImpactSurfaces: []reviewreport.ReviewReportImpactSurfaceCoverage{
				{SurfaceID: "surface-1", Status: reviewreport.ReviewReportImpactSurfaceChecked, Summary: "surface-1 was checked."},
			},
			ReviewedCandidateRisks: []reviewreport.ReviewReportCandidateRiskCoverage{
				{
					RiskID:     "risk-1",
					Status:     reviewreport.ReviewReportCandidateRiskFinding,
					Summary:    "risk-1 became finding-1.",
					FindingIDs: []string{"finding-1"},
				},
			},
		},
	}
}

func newCleanScopeCoverageForTest() *reviewreport.ReviewReportScopeCoverage {
	return &reviewreport.ReviewReportScopeCoverage{
		ReviewedImpactSurfaces: []reviewreport.ReviewReportImpactSurfaceCoverage{
			{
				SurfaceID: "surface-1",
				Status:    reviewreport.ReviewReportImpactSurfaceChecked,
				Summary:   "surface-1 was checked.",
			},
		},
		ReviewedCandidateRisks: []reviewreport.ReviewReportCandidateRiskCoverage{
			{
				RiskID:  "risk-1",
				Status:  reviewreport.ReviewReportCandidateRiskDismissed,
				Summary: "risk-1 was dismissed.",
			},
		},
	}
}

func newSaturatedReviewSaturationCheckForTest() reviewreport.ReviewSaturationCheck {
	return reviewreport.ReviewSaturationCheck{
		SchemaVersion:  reviewreport.ReviewSaturationCheckSchemaVersionV1,
		Status:         reviewreport.ReviewSaturationStatusSaturated,
		CheckedSummary: "Final report covers Pass1 surfaces and risks.",
	}
}

func mustMarshalReviewSaturationCheckForTest(t *testing.T, check reviewreport.ReviewSaturationCheck) []byte {
	t.Helper()

	data, err := json.Marshal(check)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v, want nil", err)
	}
	return data
}
