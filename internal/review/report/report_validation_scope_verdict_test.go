package report

import (
	"strings"
	"testing"
)

func TestValidateReviewReportAgainstPlanScopeCleanVerdictRequiresCleanScopeCoverage(t *testing.T) {
	tests := []struct {
		name        string
		mutate      func(*ReviewReport)
		errContains string
	}{
		{
			name: "impact surface finding",
			mutate: func(report *ReviewReport) {
				report.ScopeCoverage.ReviewedImpactSurfaces[0].Status = ReviewReportImpactSurfaceFinding
			},
		},
		{
			name: "impact surface unverified",
			mutate: func(report *ReviewReport) {
				report.ScopeCoverage.ReviewedImpactSurfaces[0].Status = ReviewReportImpactSurfaceUnverified
			},
		},
		{
			name: "impact surface residual risk",
			mutate: func(report *ReviewReport) {
				report.ScopeCoverage.ReviewedImpactSurfaces[0].Status = ReviewReportImpactSurfaceResidualRisk
			},
		},
		{
			name: "candidate risk finding",
			mutate: func(report *ReviewReport) {
				report.ScopeCoverage.ReviewedCandidateRisks[0].Status = ReviewReportCandidateRiskFinding
			},
		},
		{
			name: "candidate risk unverified",
			mutate: func(report *ReviewReport) {
				report.ScopeCoverage.ReviewedCandidateRisks[0].Status = ReviewReportCandidateRiskUnverified
			},
		},
		{
			name: "candidate risk residual risk",
			mutate: func(report *ReviewReport) {
				report.ScopeCoverage.ReviewedCandidateRisks[0].Status = ReviewReportCandidateRiskResidualRisk
			},
		},
	}

	plan := newPlanAwarePlanScopeForValidationTest()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := newPlanAwareCleanReportForValidationTest()
			tt.mutate(&report)

			err := ValidateReviewReportAgainstPlanScope(report, plan, nil)
			if err == nil {
				t.Fatal("ValidateReviewReportAgainstPlanScope() error = nil, want error")
			}
			errContains := tt.errContains
			if errContains == "" {
				errContains = `verdict "clean" requires scope_coverage`
			}
			if !strings.Contains(err.Error(), errContains) {
				t.Fatalf("ValidateReviewReportAgainstPlanScope() error = %q, want substring %q", err.Error(), errContains)
			}
		})
	}
}

func TestValidateReviewReportAgainstPlanScopeBlockedVerdictRequiresUnverifiedCoverageOrReason(t *testing.T) {
	plan := newPlanAwarePlanScopeForValidationTest()

	t.Run("unverified scope coverage is a blocked reason", func(t *testing.T) {
		report := newPlanAwareBlockedReportForValidationTest()
		report.Summary = ""
		report.ScopeCoverage.ReviewedImpactSurfaces[0].Status = ReviewReportImpactSurfaceUnverified

		if err := ValidateReviewReportAgainstPlanScope(report, plan, nil); err != nil {
			t.Fatalf("ValidateReviewReportAgainstPlanScope() error = %v, want nil", err)
		}
	})

	t.Run("existing summary blocked reason is valid", func(t *testing.T) {
		report := newPlanAwareBlockedReportForValidationTest()

		if err := ValidateReviewReportAgainstPlanScope(report, plan, nil); err != nil {
			t.Fatalf("ValidateReviewReportAgainstPlanScope() error = %v, want nil", err)
		}
	})

	t.Run("blocked without unverified coverage or reason is rejected", func(t *testing.T) {
		report := newPlanAwareBlockedReportForValidationTest()
		report.Summary = ""

		err := ValidateReviewReportAgainstPlanScope(report, plan, nil)
		if err == nil {
			t.Fatal("ValidateReviewReportAgainstPlanScope() error = nil, want error")
		}
		if !strings.Contains(err.Error(), "blocked reason") {
			t.Fatalf("ValidateReviewReportAgainstPlanScope() error = %q, want blocked reason error", err.Error())
		}
	})
}

func TestValidateReviewReportAgainstPlanScopeBlockedVerdictLinksPartialFindings(t *testing.T) {
	plan := newPlanAwarePlanScopeForValidationTest()

	t.Run("blocked partial finding requires ID for scope coverage", func(t *testing.T) {
		report := newPlanAwareBlockedReportWithRootCauseFindingForValidationTest()
		report.RootCauseGroups[0].Findings[0].ID = ""
		report.ScopeCoverage.ReviewedImpactSurfaces[0].Status = ReviewReportImpactSurfaceUnverified

		err := ValidateReviewReportAgainstPlanScope(report, plan, nil)
		if err == nil {
			t.Fatal("ValidateReviewReportAgainstPlanScope() error = nil, want error")
		}
		if !strings.Contains(err.Error(), "id must be non-empty so scope_coverage can reference it") {
			t.Fatalf("ValidateReviewReportAgainstPlanScope() error = %q, want finding ID error", err.Error())
		}
	})

	t.Run("blocked partial finding must be linked from scope coverage", func(t *testing.T) {
		report := newPlanAwareBlockedReportWithRootCauseFindingForValidationTest()
		report.ScopeCoverage.ReviewedImpactSurfaces[0].Status = ReviewReportImpactSurfaceUnverified

		err := ValidateReviewReportAgainstPlanScope(report, plan, nil)
		if err == nil {
			t.Fatal("ValidateReviewReportAgainstPlanScope() error = nil, want error")
		}
		if !strings.Contains(err.Error(), "must be referenced by scope_coverage finding_ids or new_findings_from_report_pass") {
			t.Fatalf("ValidateReviewReportAgainstPlanScope() error = %q, want linked finding error", err.Error())
		}
	})

	t.Run("blocked partial finding may be linked from a finding risk", func(t *testing.T) {
		report := newPlanAwareBlockedReportWithRootCauseFindingForValidationTest()
		report.Summary = ""
		report.ScopeCoverage.ReviewedImpactSurfaces[0].Status = ReviewReportImpactSurfaceUnverified
		report.ScopeCoverage.ReviewedCandidateRisks[0].Status = ReviewReportCandidateRiskFinding
		report.ScopeCoverage.ReviewedCandidateRisks[0].FindingIDs = []string{"finding-1"}

		if err := ValidateReviewReportAgainstPlanScope(report, plan, nil); err != nil {
			t.Fatalf("ValidateReviewReportAgainstPlanScope() error = %v, want nil", err)
		}
	})
}
