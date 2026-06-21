package report

import (
	"strings"
	"testing"
)

func TestValidateReviewReportAgainstPlanScopeRequiresCompleteScopeCoverage(t *testing.T) {
	tests := []struct {
		name        string
		report      func() ReviewReport
		errContains string
	}{
		{
			name: "missing scope coverage",
			report: func() ReviewReport {
				return newCleanReportForValidationTest(ReviewVerificationVerified)
			},
			errContains: "scope_coverage is required",
		},
		{
			name: "missing impact surface coverage",
			report: func() ReviewReport {
				report := newPlanAwareCleanReportForValidationTest()
				report.ScopeCoverage.ReviewedImpactSurfaces = nil
				return report
			},
			errContains: "missing impact surface ID",
		},
		{
			name: "missing candidate risk coverage",
			report: func() ReviewReport {
				report := newPlanAwareCleanReportForValidationTest()
				report.ScopeCoverage.ReviewedCandidateRisks = nil
				return report
			},
			errContains: "missing candidate risk ID",
		},
		{
			name: "unknown impact surface coverage",
			report: func() ReviewReport {
				report := newPlanAwareCleanReportForValidationTest()
				report.ScopeCoverage.ReviewedImpactSurfaces[0].SurfaceID = "surface-unknown"
				return report
			},
			errContains: "unknown impact surface ID",
		},
		{
			name: "unknown candidate risk coverage",
			report: func() ReviewReport {
				report := newPlanAwareCleanReportForValidationTest()
				report.ScopeCoverage.ReviewedCandidateRisks[0].RiskID = "risk-unknown"
				return report
			},
			errContains: "unknown candidate risk ID",
		},
		{
			name: "duplicate impact surface coverage",
			report: func() ReviewReport {
				report := newPlanAwareCleanReportForValidationTest()
				report.ScopeCoverage.ReviewedImpactSurfaces = append(report.ScopeCoverage.ReviewedImpactSurfaces, report.ScopeCoverage.ReviewedImpactSurfaces[0])
				return report
			},
			errContains: "duplicates impact surface ID",
		},
		{
			name: "duplicate candidate risk coverage",
			report: func() ReviewReport {
				report := newPlanAwareCleanReportForValidationTest()
				report.ScopeCoverage.ReviewedCandidateRisks = append(report.ScopeCoverage.ReviewedCandidateRisks, report.ScopeCoverage.ReviewedCandidateRisks[0])
				return report
			},
			errContains: "duplicates candidate risk ID",
		},
	}

	plan := newPlanAwarePlanScopeForValidationTest()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateReviewReportAgainstPlanScope(tt.report(), plan, nil)
			if err == nil {
				t.Fatal("ValidateReviewReportAgainstPlanScope() error = nil, want error")
			}
			if !strings.Contains(err.Error(), tt.errContains) {
				t.Fatalf("ValidateReviewReportAgainstPlanScope() error = %q, want substring %q", err.Error(), tt.errContains)
			}
		})
	}
}
