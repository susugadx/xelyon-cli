package report

import (
	"strings"
	"testing"
)

func TestValidateReviewReportAgainstPlanScopeHasFindingsRiskCoverage(t *testing.T) {
	plan := newPlanAwarePlanScopeForValidationTest()

	t.Run("risk finding with evidence-backed root cause finding is valid", func(t *testing.T) {
		report := newPlanAwareHasFindingsReportForValidationTest()
		if err := ValidateReviewReportAgainstPlanScope(report, plan, nil); err != nil {
			t.Fatalf("ValidateReviewReportAgainstPlanScope() error = %v, want nil", err)
		}
	})

	t.Run("risk finding requires finding IDs", func(t *testing.T) {
		report := newPlanAwareHasFindingsReportForValidationTest()
		report.ScopeCoverage.ReviewedCandidateRisks[0].FindingIDs = nil

		err := ValidateReviewReportAgainstPlanScope(report, plan, nil)
		if err == nil {
			t.Fatal("ValidateReviewReportAgainstPlanScope() error = nil, want error")
		}
		if !strings.Contains(err.Error(), "finding_ids must contain at least one root cause finding ID") {
			t.Fatalf("ValidateReviewReportAgainstPlanScope() error = %q, want finding_ids error", err.Error())
		}
	})

	t.Run("risk finding rejects unknown finding ID", func(t *testing.T) {
		report := newPlanAwareHasFindingsReportForValidationTest()
		report.ScopeCoverage.ReviewedCandidateRisks[0].FindingIDs = []string{"finding-unknown"}

		err := ValidateReviewReportAgainstPlanScope(report, plan, nil)
		if err == nil {
			t.Fatal("ValidateReviewReportAgainstPlanScope() error = nil, want error")
		}
		if !strings.Contains(err.Error(), "references unknown root cause finding ID") {
			t.Fatalf("ValidateReviewReportAgainstPlanScope() error = %q, want unknown finding error", err.Error())
		}
	})

	t.Run("risk finding requires evidence-backed root cause finding", func(t *testing.T) {
		report := newPlanAwareHasFindingsReportForValidationTest()
		report.RootCauseGroups[0].Findings[0].EvidenceRefs = nil

		err := ValidateReviewReportAgainstPlanScope(report, plan, nil)
		if err == nil {
			t.Fatal("ValidateReviewReportAgainstPlanScope() error = nil, want error")
		}
		if !strings.Contains(err.Error(), "evidence_refs") {
			t.Fatalf("ValidateReviewReportAgainstPlanScope() error = %q, want evidence_refs error", err.Error())
		}
	})

	t.Run("root cause finding requires ID for scope coverage", func(t *testing.T) {
		report := newPlanAwareHasFindingsReportForValidationTest()
		report.RootCauseGroups[0].Findings[0].ID = ""
		report.ScopeCoverage.ReviewedCandidateRisks[0].Status = ReviewReportCandidateRiskDismissed
		report.ScopeCoverage.ReviewedCandidateRisks[0].FindingIDs = nil

		err := ValidateReviewReportAgainstPlanScope(report, plan, nil)
		if err == nil {
			t.Fatal("ValidateReviewReportAgainstPlanScope() error = nil, want error")
		}
		if !strings.Contains(err.Error(), "id must be non-empty so scope_coverage can reference it") {
			t.Fatalf("ValidateReviewReportAgainstPlanScope() error = %q, want finding ID error", err.Error())
		}
	})

	t.Run("root cause finding must be linked from scope coverage", func(t *testing.T) {
		report := newPlanAwareHasFindingsReportForValidationTest()
		report.ScopeCoverage.ReviewedCandidateRisks[0].Status = ReviewReportCandidateRiskDismissed
		report.ScopeCoverage.ReviewedCandidateRisks[0].FindingIDs = nil

		err := ValidateReviewReportAgainstPlanScope(report, plan, nil)
		if err == nil {
			t.Fatal("ValidateReviewReportAgainstPlanScope() error = nil, want error")
		}
		if !strings.Contains(err.Error(), "must be referenced by scope_coverage finding_ids or new_findings_from_report_pass") {
			t.Fatalf("ValidateReviewReportAgainstPlanScope() error = %q, want linked finding error", err.Error())
		}
	})
}

func TestValidateReviewReportAgainstPlanScopeRejectsFindingLinksOnNonFindingScopeStatus(t *testing.T) {
	plan := newPlanAwarePlanScopeForValidationTest()

	tests := []struct {
		name   string
		mutate func(*ReviewReport)
	}{
		{
			name: "checked impact surface cannot link finding",
			mutate: func(report *ReviewReport) {
				report.ScopeCoverage.ReviewedImpactSurfaces[0].FindingIDs = []string{"finding-1"}
			},
		},
		{
			name: "unverified impact surface cannot link finding",
			mutate: func(report *ReviewReport) {
				report.ScopeCoverage.ReviewedImpactSurfaces[0].Status = ReviewReportImpactSurfaceUnverified
				report.ScopeCoverage.ReviewedImpactSurfaces[0].FindingIDs = []string{"finding-1"}
			},
		},
		{
			name: "dismissed candidate risk cannot link finding",
			mutate: func(report *ReviewReport) {
				report.ScopeCoverage.ReviewedCandidateRisks[0].Status = ReviewReportCandidateRiskDismissed
			},
		},
		{
			name: "residual risk candidate risk cannot link finding",
			mutate: func(report *ReviewReport) {
				report.ScopeCoverage.ReviewedCandidateRisks[0].Status = ReviewReportCandidateRiskResidualRisk
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := newPlanAwareHasFindingsReportForValidationTest()
			tt.mutate(&report)

			err := ValidateReviewReportAgainstPlanScope(report, plan, nil)
			if err == nil {
				t.Fatal("ValidateReviewReportAgainstPlanScope() error = nil, want error")
			}
			if !strings.Contains(err.Error(), "finding_ids must be empty when status is") {
				t.Fatalf("ValidateReviewReportAgainstPlanScope() error = %q, want finding_ids status error", err.Error())
			}
		})
	}

	t.Run("impact surface finding link can satisfy root cause linkage", func(t *testing.T) {
		report := newPlanAwareHasFindingsReportForValidationTest()
		setImpactSurfaceFindingCoverageForValidationTest(&report, "finding-1")
		report.ScopeCoverage.ReviewedCandidateRisks[0].Status = ReviewReportCandidateRiskDismissed
		report.ScopeCoverage.ReviewedCandidateRisks[0].FindingIDs = nil

		if err := ValidateReviewReportAgainstPlanScope(report, plan, nil); err != nil {
			t.Fatalf("ValidateReviewReportAgainstPlanScope() error = %v, want nil", err)
		}
	})
}

func TestValidateReviewReportAgainstPlanScopeAllowsNewReportPassFinding(t *testing.T) {
	report := newHasFindingsReportForValidationTest(ReviewVerificationVerified, ReviewVerificationVerified)
	report.RootCauseGroups[0].Findings[0].ID = "finding-new"
	report.ScopeCoverage = newCleanScopeCoverageForTest()
	report.ScopeCoverage.NewFindingsFromReportPass = []ReviewReportPassFindingCoverage{
		{
			FindingIDs:   []string{"finding-new"},
			Summary:      "Pass2 found a new issue outside the Pass1 risk list.",
			EvidenceRefs: []ReviewEvidenceRef{newFileEvidenceRefForValidationTest()},
		},
	}

	if err := ValidateReviewReportAgainstPlanScope(report, newPlanAwarePlanScopeForValidationTest(), nil); err != nil {
		t.Fatalf("ValidateReviewReportAgainstPlanScope() error = %v, want nil", err)
	}
}
