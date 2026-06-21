package report

import (
	"strings"
	"testing"

	"github.com/susugadx/xelyon-cli/internal/review/domain"
)

func TestValidateReviewReportAgainstPlanScopeRejectsCleanWithNonPassingTrustedProbe(t *testing.T) {
	plan := newValidPlanScopeForTest()
	report := newPlanAwareCleanReportForValidationTest()
	report.ProbeSummaries = newTrustedProbeSummariesForReportValidationTest(domain.ReviewProbeFailed)

	err := ValidateReviewReportAgainstPlanScope(report, plan, report.ProbeSummaries)
	if err == nil {
		t.Fatal("ValidateReviewReportAgainstPlanScope() error = nil, want clean trusted probe rejection")
	}
	if !strings.Contains(err.Error(), `verdict "clean"`) {
		t.Fatalf("ValidateReviewReportAgainstPlanScope() error = %q, want clean verdict error", err.Error())
	}
}

func TestValidateReviewReportAgainstPlanScopeRejectsCheckedScopeForNonPassingLinkedProbe(t *testing.T) {
	plan := newValidPlanScopeForTest()
	tests := []struct {
		name            string
		trustedSummary  ReviewProbeSummary
		mutateCoverage  func(*ReviewReport)
		wantErrContains string
	}{
		{
			name:           "failed linked probe checked surface",
			trustedSummary: newTrustedProbeSummaryForReportValidationTest(domain.ReviewProbeFailed),
			mutateCoverage: func(report *ReviewReport) {
				report.ScopeCoverage.ReviewedCandidateRisks[0].Status = ReviewReportCandidateRiskUnverified
			},
			wantErrContains: "reviewed_impact_surfaces",
		},
		{
			name:           "blocked linked probe dismissed risk",
			trustedSummary: newTrustedProbeSummaryForReportValidationTest(domain.ReviewProbeBlocked),
			mutateCoverage: func(report *ReviewReport) {
				report.ScopeCoverage.ReviewedImpactSurfaces[0].Status = ReviewReportImpactSurfaceUnverified
			},
			wantErrContains: "reviewed_candidate_risks",
		},
		{
			name:           "timed out linked probe dismissed risk",
			trustedSummary: newTrustedProbeSummaryForReportValidationTest(domain.ReviewProbeTimedOut),
			mutateCoverage: func(report *ReviewReport) {
				report.ScopeCoverage.ReviewedImpactSurfaces[0].Status = ReviewReportImpactSurfaceUnverified
			},
			wantErrContains: "reviewed_candidate_risks",
		},
		{
			name: "mutated linked probe dismissed risk",
			trustedSummary: ReviewProbeSummary{
				ProbeID:         "probe-1",
				Mode:            domain.ReviewProbeHostReadOnly,
				Status:          domain.ReviewProbeFailed,
				MutatedWorktree: true,
				Commands:        []ReviewProbeCommandSummary{{Command: "rg", Status: domain.ReviewProbeFailed}},
			},
			mutateCoverage: func(report *ReviewReport) {
				report.ScopeCoverage.ReviewedImpactSurfaces[0].Status = ReviewReportImpactSurfaceUnverified
			},
			wantErrContains: "reviewed_candidate_risks",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := newPlanAwareBlockedReportForValidationTest()
			report.ProbeSummaries = []ReviewProbeSummary{tt.trustedSummary}
			tt.mutateCoverage(&report)

			err := ValidateReviewReportAgainstPlanScope(report, plan, report.ProbeSummaries)
			if err == nil {
				t.Fatal("ValidateReviewReportAgainstPlanScope() error = nil, want non-passing probe scope error")
			}
			if !strings.Contains(err.Error(), tt.wantErrContains) || !strings.Contains(err.Error(), "must not be") {
				t.Fatalf("ValidateReviewReportAgainstPlanScope() error = %q, want %q must-not-be error", err.Error(), tt.wantErrContains)
			}
		})
	}
}

func TestValidateReviewReportAgainstPlanScopeRequiresPassedProbeEvidenceForDismissedNeedsProbeRisk(t *testing.T) {
	plan := newValidPlanScopeForTest()
	report := newPlanAwareCleanReportForValidationTest()
	report.ProbeSummaries = newTrustedProbeSummariesForReportValidationTest(domain.ReviewProbePassed)
	report.ScopeCoverage.ReviewedImpactSurfaces[0].EvidenceRefs = []ReviewEvidenceRef{newProbeCommandEvidenceRefForReportValidationTest("probe-1")}

	err := ValidateReviewReportAgainstPlanScope(report, plan, report.ProbeSummaries)
	if err == nil {
		t.Fatal("ValidateReviewReportAgainstPlanScope() error = nil, want risk probe evidence error")
	}
	if !strings.Contains(err.Error(), "reviewed_candidate_risks") || !strings.Contains(err.Error(), "passed linked probe") {
		t.Fatalf("ValidateReviewReportAgainstPlanScope() error = %q, want risk evidence error", err.Error())
	}
}

func TestValidateReviewReportAgainstPlanScopeRequiresPassedProbeEvidenceForCheckedNeedsProbeSurface(t *testing.T) {
	plan := newValidPlanScopeForTest()
	report := newPlanAwareCleanReportForValidationTest()
	report.ProbeSummaries = newTrustedProbeSummariesForReportValidationTest(domain.ReviewProbePassed)
	report.ScopeCoverage.ReviewedCandidateRisks[0].EvidenceRefs = []ReviewEvidenceRef{newProbeCommandEvidenceRefForReportValidationTest("probe-1")}

	err := ValidateReviewReportAgainstPlanScope(report, plan, report.ProbeSummaries)
	if err == nil {
		t.Fatal("ValidateReviewReportAgainstPlanScope() error = nil, want surface probe evidence error")
	}
	if !strings.Contains(err.Error(), "reviewed_impact_surfaces") || !strings.Contains(err.Error(), "passed linked probe") {
		t.Fatalf("ValidateReviewReportAgainstPlanScope() error = %q, want surface evidence error", err.Error())
	}
}

func TestValidateReviewReportAgainstPlanScopeAllowsDismissedAndCheckedWithPassedProbeEvidence(t *testing.T) {
	plan := newValidPlanScopeForTest()
	report := newPlanAwareCleanReportForValidationTest()
	report.ProbeSummaries = newTrustedProbeSummariesForReportValidationTest(domain.ReviewProbePassed)
	ref := newProbeCommandEvidenceRefForReportValidationTest("probe-1")
	report.ScopeCoverage.ReviewedImpactSurfaces[0].EvidenceRefs = []ReviewEvidenceRef{ref}
	report.ScopeCoverage.ReviewedCandidateRisks[0].EvidenceRefs = []ReviewEvidenceRef{ref}

	if err := ValidateReviewReportAgainstPlanScope(report, plan, report.ProbeSummaries); err != nil {
		t.Fatalf("ValidateReviewReportAgainstPlanScope() error = %v, want nil", err)
	}
}

func TestValidateReviewReportAgainstPlanScopeRequiresTrustedProbeSummaryIDs(t *testing.T) {
	plan := newNoProbePlanScopeForTest()
	tests := []struct {
		name            string
		reportSummaries []ReviewProbeSummary
		trusted         []ReviewProbeSummary
		errContains     string
	}{
		{
			name:            "count mismatch",
			reportSummaries: nil,
			trusted:         newTrustedProbeSummariesForReportValidationTest(domain.ReviewProbePassed),
			errContains:     "count",
		},
		{
			name: "id order mismatch",
			reportSummaries: []ReviewProbeSummary{
				newTrustedProbeSummaryForReportValidationTest(domain.ReviewProbePassed, "probe-2"),
				newTrustedProbeSummaryForReportValidationTest(domain.ReviewProbePassed, "probe-1"),
			},
			trusted: []ReviewProbeSummary{
				newTrustedProbeSummaryForReportValidationTest(domain.ReviewProbePassed, "probe-1"),
				newTrustedProbeSummaryForReportValidationTest(domain.ReviewProbePassed, "probe-2"),
			},
			errContains: "must match trusted probe summary ID",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := newPlanAwareCleanReportForValidationTest()
			report.ProbeSummaries = tt.reportSummaries

			err := ValidateReviewReportAgainstPlanScope(report, plan, tt.trusted)
			if err == nil {
				t.Fatal("ValidateReviewReportAgainstPlanScope() error = nil, want trusted summary mismatch")
			}
			if !strings.Contains(err.Error(), tt.errContains) {
				t.Fatalf("ValidateReviewReportAgainstPlanScope() error = %q, want %q", err.Error(), tt.errContains)
			}
		})
	}
}

func newTrustedProbeSummariesForReportValidationTest(status domain.ReviewProbeStatus) []ReviewProbeSummary {
	return []ReviewProbeSummary{newTrustedProbeSummaryForReportValidationTest(status)}
}

func newTrustedProbeSummaryForReportValidationTest(status domain.ReviewProbeStatus, ids ...string) ReviewProbeSummary {
	probeID := "probe-1"
	if len(ids) > 0 {
		probeID = ids[0]
	}
	return ReviewProbeSummary{
		ProbeID:  probeID,
		Mode:     domain.ReviewProbeHostReadOnly,
		Status:   status,
		Commands: []ReviewProbeCommandSummary{{Command: "rg", Status: status}},
	}
}

func newProbeCommandEvidenceRefForReportValidationTest(probeID string) ReviewEvidenceRef {
	return ReviewEvidenceRef{
		Kind:         ReviewEvidenceKindProbeCommand,
		ProbeID:      probeID,
		CommandIndex: ReviewCommandIndex(0),
	}
}
