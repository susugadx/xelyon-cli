package review

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func newValidReviewReportForValidationTest() ReviewReport {
	return ReviewReport{
		SchemaVersion:             ReviewReportSchemaVersionV2,
		TargetKind:                TargetCurrentChanges,
		GeneratedAt:               time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC),
		OverallVerificationStatus: ReviewVerificationBlockedOrInconclusive,
		Verdict:                   ReviewVerdictBlocked,
		Summary:                   "review did not complete",
		RootCauseGroups:           []ReviewRootCauseGroup{},
		ProbeSummaries: []ReviewProbeSummary{
			{
				ProbeID: "probe-1",
				Mode:    ReviewProbeHostReadOnly,
				Status:  ReviewProbePassed,
				Commands: []ReviewProbeCommandSummary{
					{Command: "rg", Status: ReviewProbePassed},
				},
			},
		},
	}
}

func newCleanReportForValidationTest(overallStatus ReviewVerificationStatus) ReviewReport {
	report := newValidReviewReportForValidationTest()
	report.Verdict = ReviewVerdictClean
	report.OverallVerificationStatus = overallStatus
	report.Summary = ""
	report.RootCauseGroups = nil
	return report
}

func newHasFindingsReportForValidationTest(overallStatus ReviewVerificationStatus, groupStatus ReviewVerificationStatus) ReviewReport {
	report := newValidReviewReportForValidationTest()
	report.Verdict = ReviewVerdictHasFindings
	report.OverallVerificationStatus = overallStatus
	report.Summary = ""
	report.RootCauseGroups = newRootCauseGroupsForValidationTest(groupStatus)
	return report
}

func newRootCauseGroupsForValidationTest(status ReviewVerificationStatus) []ReviewRootCauseGroup {
	return []ReviewRootCauseGroup{
		newRootCauseGroupForValidationTest("rc-1", "finding-1", status),
	}
}

func newRootCauseGroupForValidationTest(id, findingID string, status ReviewVerificationStatus) ReviewRootCauseGroup {
	return ReviewRootCauseGroup{
		ID:                 id,
		Title:              "test group",
		Severity:           ReviewGroupSeverityLow,
		VerificationStatus: status,
		FixStrategy:        "fix root cause",
		VerificationPlan:   []string{"run focused validation"},
		Findings: []ReviewFinding{
			{
				ID:    findingID,
				Title: "finding",
				EvidenceRefs: []ReviewEvidenceRef{
					newFileEvidenceRefForValidationTest(),
				},
			},
		},
	}
}

func newFileEvidenceRefForValidationTest() ReviewEvidenceRef {
	return ReviewEvidenceRef{
		Kind: ReviewEvidenceKindFile,
		Path: "internal/review/report_validation.go",
		Line: 1,
	}
}

func setCheckedSurfaceEvidenceRefForValidationTest(report *ReviewReport, ref ReviewEvidenceRef) {
	report.CheckedSurfaces = []ReviewSurfaceCoverage{
		{
			SurfaceID:    "surface-1",
			EvidenceRefs: []ReviewEvidenceRef{ref},
		},
	}
}

func newCleanScopeCoverageForTest() *ReviewReportScopeCoverage {
	return &ReviewReportScopeCoverage{
		ReviewedImpactSurfaces: []ReviewReportImpactSurfaceCoverage{
			{
				SurfaceID: "surface-1",
				Status:    ReviewReportImpactSurfaceChecked,
				Summary:   "surface-1 was checked.",
			},
		},
		ReviewedCandidateRisks: []ReviewReportCandidateRiskCoverage{
			{
				RiskID:  "risk-1",
				Status:  ReviewReportCandidateRiskDismissed,
				Summary: "risk-1 was dismissed.",
			},
		},
	}
}

func newPlanAwareCleanReportForValidationTest() ReviewReport {
	report := newCleanReportForValidationTest(ReviewVerificationVerified)
	report.ScopeCoverage = newCleanScopeCoverageForTest()
	return report
}

func newPlanAwareHasFindingsReportForValidationTest() ReviewReport {
	report := newHasFindingsReportForValidationTest(ReviewVerificationVerified, ReviewVerificationVerified)
	report.ScopeCoverage = &ReviewReportScopeCoverage{
		ReviewedImpactSurfaces: []ReviewReportImpactSurfaceCoverage{
			{SurfaceID: "surface-1", Status: ReviewReportImpactSurfaceChecked, Summary: "surface-1 was checked."},
		},
		ReviewedCandidateRisks: []ReviewReportCandidateRiskCoverage{
			{
				RiskID:     "risk-1",
				Status:     ReviewReportCandidateRiskFinding,
				Summary:    "risk-1 became finding-1.",
				FindingIDs: []string{"finding-1"},
			},
		},
	}
	return report
}

func newSaturatedReviewSaturationCheckForTest() ReviewSaturationCheck {
	return ReviewSaturationCheck{
		SchemaVersion:  ReviewSaturationCheckSchemaVersionV1,
		Status:         ReviewSaturationStatusSaturated,
		CheckedSummary: "Final report covers Pass1 surfaces and risks.",
	}
}

func mustMarshalReviewSaturationCheckForTest(t *testing.T, check ReviewSaturationCheck) []byte {
	t.Helper()

	data, err := json.Marshal(check)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v, want nil", err)
	}
	return data
}

func assertReviewReportComputedSummaryPointerForTest(t *testing.T, got *ReviewReportComputedSummary, want ReviewReportComputedSummary) {
	t.Helper()

	if got == nil {
		t.Fatal("ComputedSummary = nil, want runner computed summary")
	}
	assertReviewReportComputedSummaryValueForTest(t, *got, want)
}

func assertReviewReportComputedSummaryValueForTest(t *testing.T, got, want ReviewReportComputedSummary) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("computed summary mismatch:\n got  = %#v\n want = %#v", got, want)
	}
}
