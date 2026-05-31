package report

import (
	"strings"
	"testing"
)

func TestValidateReviewReportScopeCoverageBaseContract(t *testing.T) {
	tests := []reviewReportValidationCase{
		{
			name: "valid scope coverage with canonical finding IDs",
			report: func() ReviewReport {
				report := newHasFindingsReportForValidationTest(ReviewVerificationVerified, ReviewVerificationVerified)
				report.ScopeCoverage = &ReviewReportScopeCoverage{
					ReviewedImpactSurfaces: []ReviewReportImpactSurfaceCoverage{
						{
							SurfaceID:  "surface-1",
							Status:     ReviewReportImpactSurfaceFinding,
							Summary:    "surface-1 is linked to finding-1.",
							FindingIDs: []string{"finding-1"},
						},
					},
				}
				return report
			},
		},
		{
			name: "invalid impact surface scope status",
			report: func() ReviewReport {
				report := newValidReviewReportForValidationTest()
				report.ScopeCoverage = newCleanScopeCoverageForTest()
				report.ScopeCoverage.ReviewedImpactSurfaces[0].Status = ReviewReportImpactSurfaceStatus("unexpected")
				return report
			},
			wantErr:     true,
			errContains: "scope_coverage.reviewed_impact_surfaces[0].status",
		},
		{
			name: "invalid candidate risk scope status",
			report: func() ReviewReport {
				report := newValidReviewReportForValidationTest()
				report.ScopeCoverage = newCleanScopeCoverageForTest()
				report.ScopeCoverage.ReviewedCandidateRisks[0].Status = ReviewReportCandidateRiskStatus("unexpected")
				return report
			},
			wantErr:     true,
			errContains: "scope_coverage.reviewed_candidate_risks[0].status",
		},
		{
			name: "scope coverage evidence refs are validated",
			report: func() ReviewReport {
				report := newValidReviewReportForValidationTest()
				report.ScopeCoverage = newCleanScopeCoverageForTest()
				report.ScopeCoverage.ReviewedImpactSurfaces[0].EvidenceRefs = []ReviewEvidenceRef{{Kind: ReviewEvidenceKindFile}}
				return report
			},
			wantErr:     true,
			errContains: "scope_coverage.reviewed_impact_surfaces[0].evidence_refs[0].path",
		},
		{
			name: "scope coverage finding IDs must be canonical",
			report: func() ReviewReport {
				report := newValidReviewReportForValidationTest()
				report.ScopeCoverage = newCleanScopeCoverageForTest()
				report.ScopeCoverage.ReviewedCandidateRisks[0].FindingIDs = []string{"finding 1"}
				return report
			},
			wantErr:     true,
			errContains: "scope_coverage.reviewed_candidate_risks[0].finding_ids[0]",
		},
		{
			name: "new report pass finding entry requires finding IDs",
			report: func() ReviewReport {
				report := newValidReviewReportForValidationTest()
				report.ScopeCoverage = newCleanScopeCoverageForTest()
				report.ScopeCoverage.NewFindingsFromReportPass = []ReviewReportPassFindingCoverage{{Summary: "new finding"}}
				return report
			},
			wantErr:     true,
			errContains: "scope_coverage.new_findings_from_report_pass[0].finding_ids",
		},
	}

	runReviewReportValidationCases(t, tests)
}

func TestValidateReviewReportScopeCoverageSemanticContract(t *testing.T) {
	tests := []reviewReportValidationCase{
		{
			name: "clean verdict rejects impact surface finding status",
			report: func() ReviewReport {
				report := newCleanReportForValidationTest(ReviewVerificationVerified)
				report.ScopeCoverage = newCleanScopeCoverageForTest()
				report.ScopeCoverage.ReviewedImpactSurfaces[0].Status = ReviewReportImpactSurfaceFinding
				return report
			},
			wantErr:     true,
			errContains: `verdict "clean" requires scope_coverage.reviewed_impact_surfaces[0].status`,
		},
		{
			name: "clean verdict rejects candidate risk finding status",
			report: func() ReviewReport {
				report := newCleanReportForValidationTest(ReviewVerificationVerified)
				report.ScopeCoverage = newCleanScopeCoverageForTest()
				report.ScopeCoverage.ReviewedCandidateRisks[0].Status = ReviewReportCandidateRiskFinding
				return report
			},
			wantErr:     true,
			errContains: `verdict "clean" requires scope_coverage.reviewed_candidate_risks[0].status`,
		},
		{
			name: "non-finding status cannot link finding IDs",
			report: func() ReviewReport {
				report := newHasFindingsReportForValidationTest(ReviewVerificationVerified, ReviewVerificationVerified)
				report.ScopeCoverage = newCleanScopeCoverageForTest()
				report.ScopeCoverage.ReviewedCandidateRisks[0].FindingIDs = []string{"finding-1"}
				return report
			},
			wantErr:     true,
			errContains: "finding_ids must be empty when status is",
		},
		{
			name: "impact surface finding requires finding IDs",
			report: func() ReviewReport {
				return newImpactSurfaceFindingReportForValidationTest()
			},
			wantErr:     true,
			errContains: "scope_coverage.reviewed_impact_surfaces[0].finding_ids must contain at least one root cause finding ID",
		},
		{
			name: "impact surface finding rejects unknown finding ID",
			report: func() ReviewReport {
				return newImpactSurfaceFindingReportForValidationTest("finding-unknown")
			},
			wantErr:     true,
			errContains: "scope_coverage.reviewed_impact_surfaces[0].finding_ids[0] references unknown root cause finding ID",
		},
		{
			name: "impact surface finding requires evidence-backed root cause finding",
			report: func() ReviewReport {
				report := newBlockedImpactSurfaceFindingReportForValidationTest("finding-1")
				report.RootCauseGroups[0].Findings[0].EvidenceRefs = nil
				return report
			},
			wantErr:     true,
			errContains: `scope_coverage.reviewed_impact_surfaces[0].finding_ids[0] references root cause finding ID "finding-1" without evidence_refs`,
		},
		{
			name: "impact surface finding with evidence-backed root cause finding is valid",
			report: func() ReviewReport {
				return newImpactSurfaceFindingReportForValidationTest("finding-1")
			},
		},
		{
			name: "candidate risk finding requires finding IDs",
			report: func() ReviewReport {
				report := newHasFindingsReportForValidationTest(ReviewVerificationVerified, ReviewVerificationVerified)
				report.ScopeCoverage = newCleanScopeCoverageForTest()
				report.ScopeCoverage.ReviewedCandidateRisks[0].Status = ReviewReportCandidateRiskFinding
				return report
			},
			wantErr:     true,
			errContains: "finding_ids must contain at least one root cause finding ID",
		},
		{
			name: "root cause finding must be linked from scope coverage",
			report: func() ReviewReport {
				report := newHasFindingsReportForValidationTest(ReviewVerificationVerified, ReviewVerificationVerified)
				report.ScopeCoverage = newCleanScopeCoverageForTest()
				return report
			},
			wantErr:     true,
			errContains: "must be referenced by scope_coverage finding_ids or new_findings_from_report_pass",
		},
	}

	runReviewReportValidationCases(t, tests)
}

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

func newImpactSurfaceFindingReportForValidationTest(findingIDs ...string) ReviewReport {
	report := newHasFindingsReportForValidationTest(ReviewVerificationVerified, ReviewVerificationVerified)
	report.ScopeCoverage = newCleanScopeCoverageForTest()
	setImpactSurfaceFindingCoverageForValidationTest(&report, findingIDs...)
	return report
}

func newBlockedImpactSurfaceFindingReportForValidationTest(findingIDs ...string) ReviewReport {
	report := newBlockedReportForValidationTest(ReviewVerificationPartiallyVerified)
	report.RootCauseGroups = newRootCauseGroupsForValidationTest(ReviewVerificationVerified)
	report.ScopeCoverage = newCleanScopeCoverageForTest()
	setImpactSurfaceFindingCoverageForValidationTest(&report, findingIDs...)
	return report
}

func setImpactSurfaceFindingCoverageForValidationTest(report *ReviewReport, findingIDs ...string) {
	report.ScopeCoverage.ReviewedImpactSurfaces[0].Status = ReviewReportImpactSurfaceFinding
	report.ScopeCoverage.ReviewedImpactSurfaces[0].FindingIDs = append([]string(nil), findingIDs...)
}

func newPlanAwarePlanScopeForValidationTest() PlanScope {
	return newNoProbePlanScopeForTest()
}

func newPlanAwareCleanReportForValidationTest() ReviewReport {
	report := newCleanReportForValidationTest(ReviewVerificationVerified)
	report.ScopeCoverage = newCleanScopeCoverageForTest()
	return report
}

func newPlanAwareBlockedReportForValidationTest() ReviewReport {
	report := newBlockedReportForValidationTest(ReviewVerificationBlockedOrInconclusive)
	report.ScopeCoverage = newCleanScopeCoverageForTest()
	return report
}

func newPlanAwareBlockedReportWithRootCauseFindingForValidationTest() ReviewReport {
	report := newPlanAwareBlockedReportForValidationTest()
	report.RootCauseGroups = newRootCauseGroupsForValidationTest(ReviewVerificationVerified)
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
