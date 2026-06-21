package report

import (
	"testing"

	"github.com/susugadx/xelyon-cli/internal/review/domain"
)

func TestAuditReviewReportCoverageReportsIgnoredNonPassingLinkedProbe(t *testing.T) {
	tests := []struct {
		name   string
		status domain.ReviewProbeStatus
	}{
		{name: "failed", status: domain.ReviewProbeFailed},
		{name: "blocked", status: domain.ReviewProbeBlocked},
		{name: "timed out", status: domain.ReviewProbeTimedOut},
		{name: "mutated worktree", status: domain.ReviewProbeMutatedWorktree},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			report := newPlanAwareCleanReportForValidationTest()
			probeSummary := newTrustedProbeSummaryForReportValidationTest(tt.status)

			issues := AuditReviewReportCoverage(CoverageAuditInput{
				Plan:                  newValidPlanScopeForTest(),
				Report:                report,
				TrustedProbeSummaries: []ReviewProbeSummary{probeSummary},
			})

			assertCoverageIssueForTest(t, issues, CoverageIssueKindUnreflectedProbeOutcome, "surface-1", "")
			assertCoverageIssueForTest(t, issues, CoverageIssueKindUnreflectedProbeOutcome, "", "risk-1")
			assertCoverageIssueSeverityForTest(t, issues, CoverageIssueKindUnreflectedProbeOutcome, "surface-1", "", CoverageIssueSeverityHigh)
		})
	}
}

func TestAuditReviewReportCoverageDoesNotReportPassedLinkedProbe(t *testing.T) {
	report := newPlanAwareCleanReportForValidationTest()
	probeSummary := newTrustedProbeSummaryForReportValidationTest(domain.ReviewProbePassed)

	issues := AuditReviewReportCoverage(CoverageAuditInput{
		Plan:                  newValidPlanScopeForTest(),
		Report:                report,
		TrustedProbeSummaries: []ReviewProbeSummary{probeSummary},
	})

	assertNoCoverageIssueKindForTest(t, issues, CoverageIssueKindUnreflectedProbeOutcome)
	merged := MergeCoverageIssuesIntoSaturationCheck(newSaturatedReviewSaturationCheckForTest(), issues)
	if len(merged.AdditionalFindingCandidates) != 0 {
		t.Fatalf("AdditionalFindingCandidates = %#v, want none for passed probe", merged.AdditionalFindingCandidates)
	}
}
