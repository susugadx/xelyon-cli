package report

import (
	"reflect"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/review/domain"
)

func TestComputeReviewReportComputedSummaryCanonicalizesMutatedProbe(t *testing.T) {
	got := ComputeReviewReportComputedSummary(ReviewReport{}, []ReviewProbeSummary{
		{
			ProbeID:         "probe-1",
			Mode:            domain.ReviewProbeHostReadOnly,
			Status:          domain.ReviewProbeFailed,
			MutatedWorktree: true,
		},
	})

	assertReviewReportComputedSummaryValueForTest(t, got, ReviewReportComputedSummary{
		ProbeCount:                1,
		MutatedWorktreeProbeCount: 1,
	})
}

func TestComputeReviewReportComputedSummaryCountsScopeCoverageStatusesAndReportPassFindingIDs(t *testing.T) {
	report := ReviewReport{
		ScopeCoverage: &ReviewReportScopeCoverage{
			ReviewedImpactSurfaces: []ReviewReportImpactSurfaceCoverage{
				{SurfaceID: "surface-checked", Status: ReviewReportImpactSurfaceChecked},
				{SurfaceID: "surface-finding", Status: ReviewReportImpactSurfaceFinding},
				{SurfaceID: "surface-unverified", Status: ReviewReportImpactSurfaceUnverified},
				{SurfaceID: "surface-residual", Status: ReviewReportImpactSurfaceResidualRisk},
			},
			ReviewedCandidateRisks: []ReviewReportCandidateRiskCoverage{
				{RiskID: "risk-dismissed", Status: ReviewReportCandidateRiskDismissed},
				{RiskID: "risk-finding", Status: ReviewReportCandidateRiskFinding},
				{RiskID: "risk-unverified", Status: ReviewReportCandidateRiskUnverified},
				{RiskID: "risk-residual", Status: ReviewReportCandidateRiskResidualRisk},
			},
			NewFindingsFromReportPass: []ReviewReportPassFindingCoverage{
				{FindingIDs: []string{"finding-a", "finding-b"}},
				{FindingIDs: []string{"finding-b", "finding-c"}},
			},
		},
	}

	got := ComputeReviewReportComputedSummary(report, nil)
	assertReviewReportComputedSummaryValueForTest(t, got, ReviewReportComputedSummary{
		CheckedSurfaceCount:       1,
		FindingSurfaceCount:       1,
		UnverifiedSurfaceCount:    1,
		ResidualSurfaceCount:      1,
		CandidateRiskCount:        4,
		DismissedRiskCount:        1,
		FindingRiskCount:          1,
		UnverifiedRiskCount:       1,
		ResidualRiskCount:         1,
		NewReportPassFindingCount: 3,
	})
}

func assertReviewReportComputedSummaryValueForTest(t *testing.T, got, want ReviewReportComputedSummary) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("computed summary mismatch:\n got  = %#v\n want = %#v", got, want)
	}
}
